package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/hymkor/csvi"
	"github.com/hymkor/csvi/uncsv"
)

type typeModified int

const (
	notModified = iota
	modified
	newRow
)

func csvRowModified(csvRow *uncsv.Row) typeModified {
	bits := 0
	for _, cell := range csvRow.Cell {
		if cell.Modified() {
			bits |= 1
		}
		if len(cell.Original()) > 0 {
			bits |= 2
		}
	}
	switch bits {
	case 1:
		return newRow
	case 3:
		return modified
	}
	return notModified
}

func csvRowIsNew(row *uncsv.Row) bool {
	for _, cell := range row.Cell {
		if len(cell.Original()) > 0 {
			return false
		}
	}
	return true
}

const (
	_ANSI_CURSOR_OFF = "\x1B[?25l"
	_ANSI_CURSOR_ON  = "\x1B[?25h"
)

func ask2(msg, yes, no string, getKey func() (string, error)) (bool, error) {
	fmt.Print(msg, _ANSI_CURSOR_ON)
	for {
		answer, err := getKey()
		if err != nil {
			return false, err
		}
		if strings.Contains(yes, answer) {
			fmt.Println(answer, _ANSI_CURSOR_OFF)
			return true, nil
		}
		if strings.Contains(no, answer) {
			fmt.Println(answer, _ANSI_CURSOR_OFF)
			return false, nil
		}
	}
}

func continueOrAbort(getKey func() (string, error)) (bool, error) {
	return ask2("Continue or abort [c/a] ", "cC", "aA", getKey)
}

func askSqlAndExecute(ctx context.Context, ss *Session, getKey func() (string, error), dmlSql string) error {
	fmt.Print("\n---\n")
	fmt.Println(dmlSql)
	answer, err := ask2("Execute? [y/n] ", "yY", "nN", getKey)
	if err != nil {
		return err
	}
	if !answer {
		echoPrefix(ss.spool, "(cancel) ", dmlSql)
		return nil
	}
	err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
	if err != nil {
		return err
	}
	echo(ss.spool, dmlSql)
	return doDML(ctx, ss.tx, dmlSql, tee(os.Stdout, ss.spool))
}

func createWhere(row *uncsv.Row, columns []string, quoteFunc []func(string) (string, error), null string) (string, error) {
	var where strings.Builder
	for i, c := range row.Cell {
		if i > 0 {
			where.WriteString("\n   AND  ")
		} else {
			where.WriteString("\n WHERE  ")
		}
		if string(c.Original()) == null {
			fmt.Fprintf(&where, "%s is NULL", columns[i])
		} else {
			v, err := quoteFunc[i](string(c.Original()))
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&where, "%s = %s", columns[i], v)
		}
	}
	return where.String(), nil
}

func doEdit(ctx context.Context, ss *Session, command string, pilot CommandIn, out io.Writer) error {

	var conn canQuery
	if ss.tx == nil {
		conn = ss.conn
	} else {
		conn = ss.tx
	}

	// replace `edit ` to `select * from `
	_, tableAndWhere := cutField(command)
	query := "SELECT * FROM " + tableAndWhere

	table, _ := cutField(tableAndWhere)

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query: %[1]w (%[1]T)", err)
	}
	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	null := ss.DumpConfig.Null
	quoteFunc := make([]func(string) (string, error), 0, len(columnTypes))
	validateFunc := make([]func(string) (string, error), 0, len(columnTypes))
	for _, ct := range columnTypes {
		name := strings.ToUpper(ct.DatabaseTypeName())
		var v func(string) (string, error)
		_ct := ct
		if conv := ss.dbSpec.TryTypeNameToConv(name); conv != nil {
			quoteFunc = append(quoteFunc, conv)
			v = func(s string) (string, error) {
				if s == null {
					if nullable, ok := _ct.Nullable(); ok && !nullable {
						return "", errors.New("column is NOT NULL")
					}
					return s, nil
				}
				if _, err := conv(s); err != nil {
					return "", err
				}
				return s, nil
			}
		} else if strings.Contains(name, "INT") ||
			strings.Contains(name, "FLOAT") ||
			strings.Contains(name, "DOUBLE") ||
			name == "YEAR" ||
			strings.Contains(name, "REAL") ||
			strings.Contains(name, "SERIAL") ||
			strings.Contains(name, "NUMBER") ||
			strings.Contains(name, "NUMERIC") ||
			strings.Contains(name, "DECIMAL") {
			quoteFunc = append(quoteFunc, func(s string) (string, error) {
				return s, nil
			})
			v = func(s string) (string, error) {
				if s == null {
					if nullable, ok := _ct.Nullable(); ok && !nullable {
						return "", errors.New("column is NOT NULL")
					}
					return s, nil
				}
				if _, err := strconv.ParseFloat(s, 64); err != nil {
					return "", errors.New("not a number")
				}
				return s, nil
			}
		} else {
			quoteFunc = append(quoteFunc, func(s string) (string, error) {
				return "'" + strings.ReplaceAll(s, "'", "''") + "'", nil
			})
			v = func(s string) (string, error) {
				if s == null {
					if nullable, ok := _ct.Nullable(); ok && !nullable {
						return "", errors.New("column is NOT NULL")
					}
				}
				return s, nil
			}
		}
		validateFunc = append(validateFunc, v)
	}
	v := func(e *csvi.CellValidatedEvent) (string, error) {
		return validateFunc[e.Col](e.Text)
	}
	editResult, err := csvEdit(command, ss, v, pilot.AutoPilotForCsvi(), func(pOut io.Writer) error {
		_err := ss.DumpConfig.Dump(ctx, rows, pOut)
		rows.Close()
		return _err
	}, out)

	if err != nil && err != io.EOF {
		return err
	}
	if editResult == nil {
		return nil
	}
	const (
		Success = iota
		Failure
		AbortAll
	)
	status := Success

	editResult.Each(func(row *uncsv.Row) bool {
		var dmlSql string
		switch csvRowModified(row) {
		case notModified:
			return true
		case newRow:
			var sql strings.Builder
			fmt.Fprintf(&sql, "INSERT INTO %s VALUES\n( ", table)
			for i, c := range row.Cell {
				if i > 0 {
					sql.WriteByte(',')
				}
				if c.Text() == null {
					sql.WriteString("NULL")
				} else {
					var v string
					v, err = quoteFunc[i](c.Text())
					if err != nil {
						return false
					}
					sql.WriteString(v)
				}
			}
			sql.WriteString(")\n")
			dmlSql = sql.String()
		case modified:
			var sql strings.Builder
			sql.WriteString("UPDATE  ")
			sql.WriteString(table)

			del := "\n   SET  "

			for i, c := range row.Cell {
				if c.Modified() {
					if c.Text() == null {
						fmt.Fprintf(&sql, "%s%s = NULL ",
							del,
							columns[i])
					} else {
						var v string
						v, err = quoteFunc[i](c.Text())
						if err != nil {
							return false
						}
						fmt.Fprintf(&sql, "%s%s = %s ",
							del,
							columns[i],
							v)
					}
					del = ",\n        "
				}
			}
			var v string
			v, err = createWhere(row, columns, quoteFunc, null)
			if err != nil {
				return false
			}
			sql.WriteString(v)
			dmlSql = sql.String()
		}
		if status == Failure {
			if ans, _ := continueOrAbort(pilot.GetKey); ans {
				status = Success
			} else {
				status = AbortAll
			}
		}
		if status == AbortAll {
			echoPrefix(tee(os.Stderr, ss.spool), "(cancel) ", dmlSql)
		} else {
			err = askSqlAndExecute(ctx, ss, pilot.GetKey, dmlSql)
			if err != nil {
				fmt.Fprintln(tee(os.Stderr, ss.spool), err.Error())
				status = Failure
				err = nil
			}
		}
		return true
	})
	if err != nil {
		return err
	}
	editResult.RemovedRows(func(row *uncsv.Row) bool {
		if csvRowIsNew(row) {
			return true
		}
		if status == Failure {
			if ans, _ := continueOrAbort(pilot.GetKey); ans {
				status = Success
			} else {
				status = AbortAll
			}
		}
		var sql strings.Builder
		fmt.Fprintf(&sql, "DELETE FROM %s", table)
		var v string
		v, err = createWhere(row, columns, quoteFunc, null)
		if err != nil {
			return false
		}
		sql.WriteString(v)
		dmlSql := sql.String()
		if status == AbortAll {
			echoPrefix(tee(os.Stderr, ss.spool), "(cancel) ", dmlSql)
		} else {
			err = askSqlAndExecute(ctx, ss, pilot.GetKey, dmlSql)
			if err != nil {
				status = Failure
				err = nil
			}
		}
		return true
	})
	return err
}
