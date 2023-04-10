package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
)

func dumpRows(ctx context.Context, rows *sql.Rows, comma rune, useCRLF bool, w io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	csvWriter.Comma = comma
	csvWriter.UseCRLF = useCRLF

	if err := csvWriter.Write(columns); err != nil {
		return err
	}

	itemAny := make([]any, len(columns))
	itemStr := make([]string, len(columns))
	for i := range itemAny {
		itemAny[i] = new(any)
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
				itemStr[i] = fmt.Sprint(*p)
			}
		}
		if err := csvWriter.Write(itemStr); err != nil {
			return fmt.Errorf("(csv.Writer).Write: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("(sql.Rows) Err: %w", err)
	}
	return nil
}
