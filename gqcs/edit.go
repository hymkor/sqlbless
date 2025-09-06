package gqcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"strconv"
	"strings"

	"github.com/hymkor/csvi"
	"github.com/hymkor/csvi/uncsv"

	"github.com/hymkor/sqlbless/dbdialect"
	"github.com/hymkor/sqlbless/spread"
)

type getKeyAndSize = spread.GetKeyAndSize

type CanQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func cutField(s string) (string, string) {
	s = strings.TrimLeft(s, " \n\r\t\v")
	i := 0
	for len(s) > i && s[i] != ' ' && s[i] != '\n' && s[i] != '\r' && s[i] != '\t' && s[i] != '\v' {
		i++
	}
	return s[:i], s[i:]
}

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

func createWhere(row *uncsv.Row, columns []string, quoteFunc []func(string) (string, error), null string) (string, error) {
	var where strings.Builder
	for i, c := range row.Cell {
		if i > 0 {
			where.WriteString("\n   AND  ")
		} else {
			where.WriteString("\n WHERE  ")
		}
		if string(c.Original()) == null {
			fmt.Fprintf(&where, "%s is NULL", doubleQuoteIfNeed(columns[i]))
		} else {
			v, err := quoteFunc[i](string(c.Original()))
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&where, "%s = %s", doubleQuoteIfNeed(columns[i]), v)
		}
	}
	return where.String(), nil
}

func doubleQuoteIfNeed(s string) string {
	if strings.Contains(s, " ") {
		return `"` + s + `"`
	}
	return s
}

type Editor struct {
	*spread.Spread
	CanQuery
	Null string
	*dbdialect.DBDialect
	DML  func(ctx context.Context, sql string) error
	Dump func(context.Context, *sql.Rows, io.Writer) error
	Auto getKeyAndSize
}

func (ss *Editor) Edit(ctx context.Context, tableAndWhere string, out io.Writer) error {
	query := "SELECT * FROM " + tableAndWhere

	table, _ := cutField(tableAndWhere)

	rows, err := ss.QueryContext(ctx, query)
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
	quoteFunc := make([]func(string) (string, error), 0, len(columnTypes))
	validateFunc := make([]func(string) (string, error), 0, len(columnTypes))
	for _, ct := range columnTypes {
		name := strings.ToUpper(ct.DatabaseTypeName())
		var v func(string) (string, error)
		_ct := ct
		if conv := ss.TryTypeNameToConv(name); conv != nil {
			quoteFunc = append(quoteFunc, conv)
			v = func(s string) (string, error) {
				if s == ss.Null {
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
				if s == ss.Null {
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
				if s == ss.Null {
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

	editResult, err := ss.Spread.Edit(tableAndWhere, v, ss.Auto, func(w io.Writer) error {
		return ss.Dump(ctx, rows, w)
	}, out)

	if err != nil && err != io.EOF {
		return err
	}
	if editResult == nil {
		return nil
	}

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
				if c.Text() == ss.Null {
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
					if c.Text() == ss.Null {
						fmt.Fprintf(&sql, "%s%s = NULL ",
							del,
							doubleQuoteIfNeed(columns[i]))
					} else {
						var v string
						v, err = quoteFunc[i](c.Text())
						if err != nil {
							return false
						}
						fmt.Fprintf(&sql, "%s%s = %s ",
							del,
							doubleQuoteIfNeed(columns[i]),
							v)
					}
					del = ",\n        "
				}
			}
			var v string
			v, err = createWhere(row, columns, quoteFunc, ss.Null)
			if err != nil {
				return false
			}
			sql.WriteString(v)
			dmlSql = sql.String()
		}
		err = ss.DML(ctx, dmlSql)
		return true
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
		v, err = createWhere(row, columns, quoteFunc, ss.Null)
		if err != nil {
			return false
		}
		sql.WriteString(v)
		err = ss.DML(ctx, sql.String())
		return true
	})
	return err
}
