package dialect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrColumnNameNotFound = errors.New("column name not found")
)

type CanQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func findColumn(columnName string, columns []string) int {
	for i, name := range columns {
		if strings.EqualFold(name, columnName) {
			return i
		}
	}
	return -1
}

func queryOneColumn(ctx context.Context, conn CanQuery, sqlStr, columnName string, args ...any) ([]string, error) {
	rows, err := conn.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	if !rows.Next() {
		return result, nil
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	tablePosition := findColumn(columnName, columns)
	if tablePosition < 0 {
		return nil, fmt.Errorf("%s: %w", columnName, ErrColumnNameNotFound)
	}
	values := make([]any, len(columns))
	for i := 0; i < len(columns); i++ {
		values[i] = &sql.NullString{}
	}
	uniq := map[string]struct{}{}
	for {
		err := rows.Scan(values...)
		if err != nil {
			return nil, err
		}
		if p, ok := values[tablePosition].(*sql.NullString); ok && p.Valid {
			if _, ok := uniq[p.String]; !ok {
				result = append(result, p.String)
				uniq[p.String] = struct{}{}
			}
		}
		if !rows.Next() {
			return result, rows.Err()
		}
	}
}

func (e *Entry) Tables(ctx context.Context, conn CanQuery) ([]string, error) {
	return queryOneColumn(ctx, conn, e.SqlForTab, e.TableField)
}

func (e *Entry) Columns(ctx context.Context, conn CanQuery, table string) ([]string, error) {
	sqlStr := strings.ReplaceAll(e.SqlForDesc, "{table_name}", table)
	return queryOneColumn(ctx, conn, sqlStr, e.ColumnField, table)
}
