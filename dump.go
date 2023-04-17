package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

var (
	flagNullString     = flag.String("null", "<NULL>", "Set a string representing NULL")
	flagFieldSeperator = flag.String("fs", ",", "Set field separator")
)

type lfToCrlf struct{}

func (t lfToCrlf) Reset() {}

func (f lfToCrlf) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for _, c := range src {
		if c == '\n' {
			if len(dst) < 2 {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[0] = '\r'
			dst[1] = '\n'
			dst = dst[2:]
			nDst += 2
		} else {
			if len(dst) < 1 {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[0] = c
			dst = dst[1:]
			nDst++
		}
		nSrc++
	}
	return nDst, nSrc, nil
}

type writeMultiCloser struct {
	io.Writer
	c []io.Closer
}

func (m *writeMultiCloser) Close() error {
	for _, p := range m.c {
		p.Close()
	}
	return nil
}

func lfToCrlfWriter(src io.WriteCloser) io.WriteCloser {
	w := transform.NewWriter(src, lfToCrlf{})
	return &writeMultiCloser{
		Writer: w,
		c:      []io.Closer{w, src},
	}
}

func dumpRows(ctx context.Context, rows *sql.Rows, w io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	csvWriter.Comma, _ = utf8.DecodeRuneInString(*flagFieldSeperator)
	csvWriter.UseCRLF = false

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
				if *p == nil {
					itemStr[i] = *flagNullString
					continue
				}
				if b, ok := (*p).([]byte); ok && utf8.Valid(b) {
					itemStr[i] = string(b)
					continue
				}
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
