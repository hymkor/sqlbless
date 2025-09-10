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

func dump(ctx context.Context, rows Source, conv func(int, *sql.ColumnType, sql.NullString) string, debug bool, csvw *csv.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}

	if err := csvw.Write(columns); err != nil {
		return err
	}

	n := len(columns)
	refs := make([]any, n)
	data := make([]any, n)
	for i := 0; i < n; i++ {
		refs[i] = &data[i]
	}
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
		for i, v := range data {
			var ns sql.NullString
			if stamp, ok := v.(time.Time); ok {
				ns.String = stamp.Format("2006-01-02 15:04:05.999999999 -07:00")
				ns.Valid = true
			} else if b, ok := v.([]byte); ok {
				ns.String = string(b)
				ns.Valid = true
			} else if v != nil {
				ns.String = fmt.Sprint(v)
				ns.Valid = true
			}
			strs[i] = conv(i, columnTypes[i], ns)
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

type Config struct {
	Comma     rune
	UseCRLF   bool
	Null      string
	Debug     bool
	Conv      func(int, *sql.ColumnType, sql.NullString) string
	AutoClose bool
}

func (cfg Config) defaultConv(_ int, _ *sql.ColumnType, v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return cfg.Null
}

func (cfg Config) Dump(ctx context.Context, rows Source, w io.Writer) error {
	csvw := csv.NewWriter(w)
	defer csvw.Flush()

	csvw.Comma = cfg.Comma
	csvw.UseCRLF = cfg.UseCRLF

	conv := cfg.defaultConv
	if cfg.Conv != nil {
		conv = cfg.Conv
	}
	if cfg.AutoClose {
		defer rows.Close()
	}
	return dump(ctx, rows, conv, cfg.Debug, csvw)
}
