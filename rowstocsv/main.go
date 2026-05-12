package rowstocsv

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

type Source interface {
	Close() error
	ColumnTypes() ([]*sql.ColumnType, error)
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...any) error
}

func anyToNullString(v any) (string, bool) {
	if stamp, ok := v.(time.Time); ok {
		return stamp.Format("2006-01-02 15:04:05.999999999 -07:00"), true
	} else if b, ok := v.([]byte); ok {
		return string(b), true
	} else if v != nil {
		return fmt.Sprint(v), true
	}
	return "", false
}

func makeBuffers[T any](n int) ([]any, []T) {
	refs := make([]any, n)
	data := make([]T, n)
	for i := 0; i < n; i++ {
		refs[i] = &data[i]
	}
	return refs, data
}

func dump(ctx context.Context, rows Source, conv func(int, *sql.ColumnType, any) (string, bool), null string, debug bool, write func([]string) error) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	if err := write(columns); err != nil {
		return err
	}

	n := len(columns)
	refs, data := makeBuffers[any](n)
	strs := make([]string, len(columns))

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		for i := 0; i < n; i++ {
			columnTypes[i] = nil
		}
	} else if debug {
		for i, c := range columnTypes {
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
		write(strs)
	}

	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rows.Scan(refs...); err != nil {
			return err
		}
		for i, v := range data {
			if conv != nil {
				s, ok := conv(i, columnTypes[i], v)
				if ok {
					strs[i] = s
					continue
				}
			}
			if s, ok := anyToNullString(v); ok {
				strs[i] = s
			} else {
				strs[i] = null
			}
		}
		if err := write(strs); err != nil {
			return fmt.Errorf("(csv.Writer).Write: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("(sql.Rows) Err: %w", err)
	}
	return nil
}

type Config struct {
	Comma     rune
	UseCRLF   bool
	Null      string
	Debug     bool
	Conv      func(int, *sql.ColumnType, any) (string, bool)
	AutoClose bool
}

func (cfg Config) Dump(ctx context.Context, rows Source, w io.Writer) error {
	csvw := csv.NewWriter(w)
	defer csvw.Flush()

	csvw.Comma = cfg.Comma
	csvw.UseCRLF = cfg.UseCRLF

	if cfg.AutoClose {
		defer rows.Close()
	}
	return dump(ctx, rows, cfg.Conv, cfg.Null, cfg.Debug, csvw.Write)
}
