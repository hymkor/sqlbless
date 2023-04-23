package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"
)

type RowToCsv struct {
	Comma     rune
	UseCRLF   bool
	Null      string
	PrintType bool
}

func (cfg RowToCsv) Dump(ctx context.Context, rows *sql.Rows, w io.Writer) error {
	csvw := csv.NewWriter(w)
	defer csvw.Flush()

	csvw.Comma = cfg.Comma
	csvw.UseCRLF = cfg.UseCRLF

	return rowsToCsv(ctx, rows, cfg.Null, cfg.PrintType, csvw)
}

func rowsToCsv(ctx context.Context, rows *sql.Rows, null string, printType bool, csvw *csv.Writer) error {
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
			itemStr[i] = c.DatabaseTypeName() + "(" + c.ScanType().String() + ")"
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
		for i, a := range itemAny {
			if p, ok := a.(*any); ok {
				if *p == nil {
					itemStr[i] = null
					continue
				}
				if b, ok := (*p).([]byte); ok && utf8.Valid(b) {
					itemStr[i] = string(b)
					continue
				}
				itemStr[i] = fmt.Sprint(*p)
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
