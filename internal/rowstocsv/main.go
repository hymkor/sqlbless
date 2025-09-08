package rowstocsv

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

type RowsInterface interface {
	Close() error
	ColumnTypes() ([]*sql.ColumnType, error)
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...any) error
}

func scanAnyAll(rows interface{ Scan(...any) error }, n int) ([]any, error) {
	refs := make([]any, n)
	data := make([]any, n)
	for i := 0; i < n; i++ {
		refs[i] = &data[i]
	}
	if err := rows.Scan(refs...); err != nil {
		return nil, err
	}
	return data, nil
}

func rowsToCsv(ctx context.Context, rows RowsInterface, null, timeLayout string, printType bool, csvw *csv.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	if err := csvw.Write(columns); err != nil {
		return err
	}

	itemStr := make([]string, len(columns))

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
		itemAny, err := scanAnyAll(rows, len(columns))
		if err != nil {
			return err
		}
		if printType {
			for i, v := range itemAny {
				itemStr[i] = fmt.Sprintf("%T", v)
			}
			csvw.Write(itemStr)
			printType = false
		}
		for i, v := range itemAny {
			if v == nil {
				itemStr[i] = null
			} else if tm, ok := v.(sql.NullTime); ok {
				if tm.Valid {
					itemStr[i] = tm.Time.Format(timeLayout)
				} else {
					itemStr[i] = null
				}
			} else if tm, ok := v.(time.Time); ok {
				itemStr[i] = tm.Format(timeLayout)
			} else if b, ok := v.([]byte); ok && utf8.Valid(b) {
				itemStr[i] = string(b)
			} else {
				itemStr[i] = fmt.Sprint(v)
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

func (cfg RowToCsv) Dump(ctx context.Context, rows RowsInterface, w io.Writer) error {
	csvw := csv.NewWriter(w)
	defer csvw.Flush()

	csvw.Comma = cfg.Comma
	csvw.UseCRLF = cfg.UseCRLF

	return rowsToCsv(ctx, rows, cfg.Null, cfg.TimeLayout, cfg.PrintType, csvw)
}
