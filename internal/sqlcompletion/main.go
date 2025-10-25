package sqlcompletion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/nyaosorg/go-readline-ny/completion"

	"github.com/hymkor/sqlbless/dialect"
)

var (
	ColumnNameNotFound = errors.New("column name not found")
)

type CanQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type completeType struct {
	Conn        CanQuery
	SqlForTab   string
	SqlForDesc  string
	TableField  string
	ColumnField string
	tableCache  []string
	columnCache map[string][]string
}

func getSqlCommands() []string {
	return []string{
		"alter",
		"commit",
		"delete",
		"desc",
		"drop",
		"edit",
		"exit",
		"history",
		"insert",
		"quit",
		"rem",
		"rollback",
		"select",
		"spool",
		"start",
		"truncate",
		"update",
	}
}

func (C *completeType) getCandidates(fields []string) ([]string, []string) {
	candidates := getSqlCommands
	tableListNow := false
	tableNameInline := []string{}
	lastKeywordAt := 0
	var nextKeyword []string
	for i, word := range fields {
		if strings.EqualFold(word, "from") || strings.EqualFold(word, "edit") {
			tableListNow = true
			lastKeywordAt = i
			nextKeyword = []string{"where"}
			candidates = func() []string {
				return C.tables()
			}
		} else if strings.EqualFold(word, "desc") || strings.EqualFold(word, "\\D") || strings.EqualFold(word, "table") {
			tableListNow = true
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				return C.tables()
			}
		} else if strings.EqualFold(word, "set") {
			tableListNow = false
			lastKeywordAt = i
			nextKeyword = []string{"where"}
			candidates = func() []string {
				return C.columns(tableNameInline)
			}
		} else if strings.EqualFold(word, "update") {
			tableListNow = true
			lastKeywordAt = i
			nextKeyword = []string{"set"}
			candidates = func() []string {
				return C.tables()
			}
		} else if strings.EqualFold(word, "delete") {
			tableListNow = true
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				return []string{"from"}
			}
		} else if strings.EqualFold(word, "select") {
			tableListNow = false
			lastKeywordAt = i
			nextKeyword = []string{"from"}
			candidates = func() []string {
				return C.columns(tableNameInline)
			}
		} else if strings.EqualFold(word, "drop") || strings.EqualFold(word, "truncate") {
			tableListNow = false
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				return []string{"table"}
			}
		} else if strings.EqualFold(word, "where") {
			tableListNow = false
			lastKeywordAt = i
			nextKeyword = []string{"and", "or"}
			candidates = func() []string {
				return C.columns(tableNameInline)
			}
		} else if strings.EqualFold(word, "start") || strings.EqualFold(word, "host") {
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				v, _ := completion.PathComplete(fields[:i+1])
				return v
			}
		} else {
			if tableListNow && i < len(fields)-1 {
				tableNameInline = append(tableNameInline, word)
			}
		}
	}
	result := candidates()
	if lastKeywordAt < len(fields)-2 && nextKeyword != nil {
		result = append(result, nextKeyword...)
	}
	return result, result
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

func (C *completeType) tables() []string {
	if len(C.tableCache) <= 0 {
		C.tableCache, _ = queryOneColumn(context.TODO(), C.Conn, C.SqlForTab, C.TableField)
	}
	return C.tableCache
}

func (C *completeType) columns(tables []string) (result []string) {
	if C.columnCache == nil {
		C.columnCache = map[string][]string{}
	}
	ctx := context.TODO()
	for _, tableName := range tables {
		if tableName == "," || tableName == "" {
			continue
		}
		values, ok := C.columnCache[tableName]
		if !ok {
			sqlStr := strings.ReplaceAll(C.SqlForDesc, "{table_name}", tableName)
			values, _ = queryOneColumn(ctx, C.Conn, sqlStr, C.ColumnField, tableName)
			C.columnCache[tableName] = values
		}
		result = append(result, values...)
	}
	return
}

func New(d *dialect.Entry, c CanQuery) func([]string) ([]string, []string) {
	completer := &completeType{
		Conn:        c,
		SqlForTab:   d.SqlForTab,
		SqlForDesc:  d.SqlForDesc,
		TableField:  d.TableField,
		ColumnField: d.ColumnField,
	}
	return completer.getCandidates
}
