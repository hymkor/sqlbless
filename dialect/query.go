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

func (e *Entry) SqlToQueryTables() string {
	return e.SQLForTables
}

// BuildSQLForColumns returns the SQL statement used to retrieve the list of columns
// for the given table name. The placeholder "{table_name}" in the template will be replaced.
func (e *Entry) BuildSQLForColumns(table string) string {
	return strings.ReplaceAll(e.SQLForColumns, "{table_name}", table)
}

func (e *Entry) Tables(ctx context.Context, conn CanQuery) ([]string, error) {
	return queryOneColumn(ctx, conn, e.SqlToQueryTables(), e.TableNameField)
}

func (e *Entry) Columns(ctx context.Context, conn CanQuery, table string) ([]string, error) {
	return queryOneColumn(ctx, conn, e.BuildSQLForColumns(table), e.ColumnNameField, table)
}
