package rowstocsv

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
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

func rowsToCsv(ctx context.Context, rows RowsInterface, null, timeLayout string, printType bool, csvw *csv.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	if err := csvw.Write(columns); err != nil {
		return err
	}

	n := len(columns)
	refs := make([]any, n)
	nstrs := make([]sql.NullString, n)
	for i := 0; i < n; i++ {
		refs[i] = &nstrs[i]
	}
	strs := make([]string, len(columns))

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
				strs[i] = buffer.String()
			} else {
				strs[i] = ""
			}

		}
		csvw.Write(strs)
	}

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := rows.Scan(refs...); err != nil {
			return err
		}
		for i, v := range nstrs {
			if v.Valid {
				strs[i] = v.String
			} else {
				strs[i] = null
			}
		}
		if err := csvw.Write(strs); err != nil {
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
