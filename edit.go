package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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

func askSqlAndExecute(ctx context.Context, ss *Session, getKey func() (string, error), dmlSql string) error {
	fmt.Println(dmlSql)
	fmt.Print("Execute? [y/n] ", _ANSI_CURSOR_ON)
	answer, err := getKey()
	fmt.Println(answer, _ANSI_CURSOR_OFF)
	if err != nil {
		return err
	}
	if answer == "y" || answer == "Y" {
		err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
		if err != nil {
			return err
		}
		echo(ss.spool, dmlSql)
		err = doDML(ctx, ss.tx, dmlSql, tee(os.Stdout, ss.spool))
		if err != nil {
			return err
		}
	} else {
		echoPrefix(ss.spool, "(cancel) ", dmlSql)
	}
	return nil
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

func doEdit(ctx context.Context, ss *Session, command string, pilot CommandIn, out, spool io.Writer) error {

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
	_rows := rowsHasNext(rows)
	if _rows == nil {
		rows.Close()
		return fmt.Errorf("data not found")
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
	quoteFunc := make([]func(string) (string, error), 0, len(columnTypes))
	for _, ct := range columnTypes {
		name := strings.ToUpper(ct.DatabaseTypeName())
		if conv := ss.dbSpec.TryTypeNameToConv(name); conv != nil {
			quoteFunc = append(quoteFunc, conv)
		} else if strings.Contains(name, "INT") ||
			strings.Contains(name, "NUMBER") ||
			strings.Contains(name, "NUMERIC") ||
			strings.Contains(name, "DECIMAL") {
			quoteFunc = append(quoteFunc, func(s string) (string, error) {
				return s, nil
			})
		} else {
			quoteFunc = append(quoteFunc, func(s string) (string, error) {
				return "'" + strings.ReplaceAll(s, "'", "''") + "'", nil
			})
		}
	}
	editResult, err := csvEdit(command, ss.DumpConfig.Comma, pilot.AutoPilotForCsvi(), func(pOut io.Writer) error {
		_err := ss.DumpConfig.Dump(ctx, _rows, pOut)
		rows.Close()
		return _err
	}, out, spool)

	if err != nil && err != io.EOF {
		return err
	}
	if editResult == nil {
		return nil
	}
	null := ss.DumpConfig.Null
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
		err = askSqlAndExecute(ctx, ss, pilot.GetKey, dmlSql)
		return err == nil
	})
	if err != nil {
		return err
	}
	editResult.RemovedRows(func(row *uncsv.Row) bool {
		if csvRowIsNew(row) {
			return true
		}
		var sql strings.Builder
		fmt.Fprintf(&sql, "DELETE FROM %s", table)
		var v string
		v, err = createWhere(row, columns, quoteFunc, null)
		if err != nil {
			return false
		}
		sql.WriteString(v)
		err = askSqlAndExecute(ctx, ss, pilot.GetKey, sql.String())
		return err == nil
	})
	return err
}
