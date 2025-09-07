package sqlbless

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

type RowToCsv struct {
	Comma      rune
	UseCRLF    bool
	Null       string
	PrintType  bool
	TimeLayout string
}

type _RowsI interface {
	Close() error
	ColumnTypes() ([]*sql.ColumnType, error)
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...any) error
}

type unreadRows struct {
	*sql.Rows
	unread bool
}

func rowsHasNext(r *sql.Rows) (*unreadRows, bool) {
	if !r.Next() {
		return nil, false
	}
	return &unreadRows{
		Rows:   r,
		unread: true,
	}, true
}

func (r *unreadRows) Next() bool {
	if r.unread {
		r.unread = false
		return true
	}
	return r.Rows.Next()
}

func (cfg RowToCsv) Dump(ctx context.Context, rows _RowsI, w io.Writer) error {
	csvw := csv.NewWriter(w)
	defer csvw.Flush()

	csvw.Comma = cfg.Comma
	csvw.UseCRLF = cfg.UseCRLF

	return rowsToCsv(ctx, rows, cfg.Null, cfg.TimeLayout, cfg.PrintType, csvw)
}

func rowsToCsv(ctx context.Context, rows _RowsI, null, timeLayout string, printType bool, csvw *csv.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	if err := csvw.Write(columns); err != nil {
		return err
	}

	itemAny := make([]any, len(columns))
	itemStr := make([]string, len(columns))
	for i := range itemAny {
		itemAny[i] = new(any)
	}

	if printType {
		ct, err := rows.ColumnTypes()
		if err != nil {
			return err
		}
		for i, c := range ct {
			if c != nil {
				var buffer strings.Builder
				buffer.WriteString(c.DatabaseTypeName())
				if st := c.ScanType(); st != nil {
					buffer.WriteByte('(')
					buffer.WriteString(st.String())
					buffer.WriteByte(')')
				}
				itemStr[i] = buffer.String()
			} else {
				itemStr[i] = ""
			}

		}
		csvw.Write(itemStr)
	}

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := rows.Scan(itemAny...); err != nil {
			return fmt.Errorf("(sql.Rows) Scan: %w", err)
		}
		if printType {
			for i, a := range itemAny {
				if p, ok := a.(*any); ok {
					itemStr[i] = fmt.Sprintf("%T", *p)
				} else {
					itemStr[i] = ""
				}
			}
			csvw.Write(itemStr)
			printType = false
		}
		for i, a := range itemAny {
			if p, ok := a.(*any); ok {
				if *p == nil {
					itemStr[i] = null
					continue
				}
				if tm, ok := (*p).(sql.NullTime); ok {
					if tm.Valid {
						itemStr[i] = tm.Time.Format(timeLayout)
					} else {
						itemStr[i] = null
					}
					continue
				}
				if tm, ok := (*p).(time.Time); ok {
					itemStr[i] = tm.Format(timeLayout)
					continue
				}
				if b, ok := (*p).([]byte); ok && utf8.Valid(b) {
					itemStr[i] = string(b)
					continue
				}
				itemStr[i] = fmt.Sprint(*p)
			} else {
				itemStr[i] = ""
			}
		}
		if err := csvw.Write(itemStr); err != nil {
			return fmt.Errorf("(csv.Writer).Write: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("(sql.Rows) Err: %w", err)
	}
	return nil
}
