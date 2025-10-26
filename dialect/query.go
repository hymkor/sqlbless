package dialect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ColumnNameNotFound = errors.New("column name not found")
)

type CanQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func queryOneColumn(ctx context.Context, conn CanQuery, sqlStr, columnName string, args ...any) ([]string, error) {
	rows, err := conn.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	uniq := map[string]struct{}{}
	tablePosition := -1
	var values []any
	var result []string
	for rows.Next() {
		if values == nil {
			columns, err := rows.Columns()
			if err != nil {
				return nil, err
			}
			for i, name := range columns {
				if strings.EqualFold(name, columnName) {
					tablePosition = i
					break
				}
			}
			if tablePosition < 0 {
				return nil, fmt.Errorf("%s: %w", columnName, ColumnNameNotFound)
			}
			nFields := len(columns)
			values = make([]any, nFields)
			for i := range values {
				values[i] = &sql.NullString{}
			}
		}
		err := rows.Scan(values...)
		if err != nil {
			return nil, err
			break
		}
		if p, ok := values[tablePosition].(*sql.NullString); ok && p.Valid {
			if _, ok := uniq[p.String]; !ok {
				result = append(result, p.String)
				uniq[p.String] = struct{}{}
			}
		}
	}
	return result, nil
}

func (e *Entry) Tables(ctx context.Context, conn CanQuery) ([]string, error) {
	return queryOneColumn(ctx, conn, e.SqlForTab, e.TableField)
}

func (e *Entry) Columns(ctx context.Context, conn CanQuery, table string) ([]string, error) {
	sqlStr := strings.ReplaceAll(e.SqlForDesc, "{table_name}", table)
	return queryOneColumn(ctx, conn, sqlStr, e.ColumnField, table)
}
