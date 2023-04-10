package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
)

func dumpRows(ctx context.Context, rows *sql.Rows, fs, rs string, w io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}
	item := make([]any, len(columns))
	for i, name := range columns {
		item[i] = new(any)
		if i > 0 {
			io.WriteString(w, fs)
		}
		io.WriteString(w, name)
	}
	io.WriteString(w, rs)

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := rows.Scan(item...); err != nil {
			return fmt.Errorf("(sql.Rows) Scan: %w", err)
		}
		for i, item1 := range item {
			if i > 0 {
				io.WriteString(w, fs)
			}
			if p, ok := item1.(*any); ok {
				fmt.Fprint(w, *p)
			}
		}
		io.WriteString(w, rs)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("(sql.Rows) Err: %w", err)
	}
	return nil
}
