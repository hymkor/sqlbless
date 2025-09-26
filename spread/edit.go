package spread

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

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/misc"
	"github.com/hymkor/sqlbless/rowstocsv"
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

func createWhere(row *uncsv.Row, columns []string, quoteFunc []func(string) (any, error), null string, holder dialect.PlaceHolder) (string, error) {
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
			c := doubleQuoteIfNeed(columns[i])
			if h, ok := holder.(interface{ NormalizeColumnForWhere(any, string) string }); ok {
				c = h.NormalizeColumnForWhere(v, c)
			}
			fmt.Fprintf(&where, "%s = %s", c, holder.Make(v))
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
	*Viewer
	*dialect.Entry
	Query func(context.Context, string, ...any) (*sql.Rows, error)
	Exec  func(context.Context, string, ...any) (sql.Result, error)
	Auto  GetKeyAndSize
}

type placeHolder struct {
	maker  func(int) string
	values []any
}

func newPlaceHolder(maker func(int) string) *placeHolder {
	return &placeHolder{maker: maker}
}

func (ph *placeHolder) Make(value any) string {
	s := ph.maker(len(ph.values))
	ph.values = append(ph.values, value)
	return s
}

func (ph *placeHolder) Values() []any {
	result := ph.values
	ph.values = ph.values[:0]
	return result
}

func (editor *Editor) Edit(ctx context.Context, tableAndWhere string, termOut io.Writer) error {
	query := "SELECT * FROM " + tableAndWhere

	table, _ := misc.CutField(tableAndWhere)

	rows, err := editor.Query(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	quoteFunc := make([]func(string) (any, error), 0, len(columnTypes))
	validateFunc := make([]func(string) (string, error), 0, len(columnTypes))
	for _, ct := range columnTypes {
		name := strings.ToUpper(ct.DatabaseTypeName())
		var v func(string) (string, error)
		_ct := ct
		if conv := editor.TypeToConv(name); conv != nil {
			quoteFunc = append(quoteFunc, conv)
			v = func(s string) (string, error) {
				if s == editor.Null || s == "" {
					if nullable, ok := _ct.Nullable(); ok && !nullable {
						return "", errors.New("column is NOT NULL")
					}
					return editor.Null, nil
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
			quoteFunc = append(quoteFunc, func(s string) (any, error) {
				if strings.ContainsRune(s, '.') {
					if v, err := strconv.ParseFloat(s, 64); err == nil {
						return v, nil
					}
				} else if v, err := strconv.ParseInt(s, 0, 64); err == nil {
					return v, nil
				}
				return s, nil
			})
			v = func(s string) (string, error) {
				if s == editor.Null || s == "" {
					if nullable, ok := _ct.Nullable(); ok && !nullable {
						return "", errors.New("column is NOT NULL")
					}
					return editor.Null, nil
				}
				if _, err := strconv.ParseFloat(s, 64); err != nil {
					return "", errors.New("not a number")
				}
				return s, nil
			}
		} else {
			quoteFunc = append(quoteFunc, func(s string) (any, error) {
				return s, nil
			})
			v = func(s string) (string, error) {
				if s == editor.Null {
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

	changes, err := editor.Viewer.edit(tableAndWhere, v, editor.Auto, func(w io.Writer) error {
		err := rowstocsv.Config{
			Null:      editor.Viewer.Null,
			Comma:     rune(editor.Viewer.Comma),
			AutoClose: true,
		}.Dump(ctx, rows, w)
		rows = nil
		return err
	}, termOut)

	if err != nil && err != io.EOF {
		return err
	}
	if changes == nil {
		return nil
	}

	changes.Each(func(row *uncsv.Row) bool {
		holder := editor.PlaceHolder
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
				if c.Text() == editor.Null {
					sql.WriteString("NULL")
				} else {
					var v any
					v, err = quoteFunc[i](c.Text())
					if err != nil {
						return false
					}
					sql.WriteString(holder.Make(v))
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
					if c.Text() == editor.Null {
						fmt.Fprintf(&sql, "%s%s = NULL ",
							del,
							doubleQuoteIfNeed(columns[i]))
					} else {
						var v any
						v, err = quoteFunc[i](c.Text())
						if err != nil {
							return false
						}
						fmt.Fprintf(&sql, "%s%s = %s ",
							del,
							doubleQuoteIfNeed(columns[i]),
							holder.Make(v))
					}
					del = ",\n        "
				}
			}
			var v string
			v, err = createWhere(row, columns, quoteFunc, editor.Null, holder)
			if err != nil {
				return false
			}
			sql.WriteString(v)
			dmlSql = sql.String()
		}
		_, err = editor.Exec(ctx, dmlSql, holder.Values()...)
		return true
	})
	if err != nil {
		return err
	}
	changes.RemovedRows(func(row *uncsv.Row) bool {
		if csvRowIsNew(row) {
			return true
		}
		holder := editor.PlaceHolder
		var sql strings.Builder
		fmt.Fprintf(&sql, "DELETE FROM %s", table)
		var v string
		v, err = createWhere(row, columns, quoteFunc, editor.Null, holder)
		if err != nil {
			return false
		}
		sql.WriteString(v)
		_, err = editor.Exec(ctx, sql.String(), holder.Values())
		return true
	})
	return err
}
