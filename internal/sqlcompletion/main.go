package sqlcompletion

import (
	"context"
	"strings"

	"github.com/nyaosorg/go-readline-ny/completion"

	"github.com/hymkor/sqlbless/dialect"
)

type completeType struct {
	Conn        dialect.CanQuery
	Dialect     *dialect.Entry
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
		"savepoint",
		"select",
		"spool",
		"start",
		"truncate",
		"update",
	}
}

func (C *completeType) getCandidates(ctx context.Context, fields []string) ([]string, []string) {
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
				return C.tables(ctx)
			}
		} else if strings.EqualFold(word, "desc") || strings.EqualFold(word, "\\D") || strings.EqualFold(word, "table") {
			tableListNow = true
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				return C.tables(ctx)
			}
		} else if strings.EqualFold(word, "set") {
			tableListNow = false
			lastKeywordAt = i
			nextKeyword = []string{"where"}
			candidates = func() []string {
				return C.columns(ctx, tableNameInline)
			}
		} else if strings.EqualFold(word, "update") {
			tableListNow = true
			lastKeywordAt = i
			nextKeyword = []string{"set"}
			candidates = func() []string {
				return C.tables(ctx)
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
				return C.columns(ctx, tableNameInline)
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
				return C.columns(ctx, tableNameInline)
			}
		} else if strings.EqualFold(word, "start") || strings.EqualFold(word, "host") {
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				v, _ := completion.PathComplete(fields[:i+1])
				return v
			}
		} else if strings.EqualFold(word, "rollback") {
			tableListNow = false
			lastKeywordAt = i
			nextKeyword = nil
			candidates = func() []string {
				return []string{"to", "transaction"}
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

func (C *completeType) tables(ctx context.Context) []string {
	if len(C.tableCache) <= 0 {
		C.tableCache, _ = C.Dialect.FetchTables(ctx, C.Conn)
	}
	return C.tableCache
}

func (C *completeType) columns(ctx context.Context, tables []string) (result []string) {
	if C.columnCache == nil {
		C.columnCache = map[string][]string{}
	}
	for _, tableName := range tables {
		if tableName == "," || tableName == "" {
			continue
		}
		values, ok := C.columnCache[tableName]
		if !ok {
			values, _ = C.Dialect.FetchColumns(ctx, C.Conn, tableName)
			C.columnCache[tableName] = values
		}
		result = append(result, values...)
	}
	return
}

func New(d *dialect.Entry, c dialect.CanQuery) func(context.Context, []string) ([]string, []string) {
	completer := &completeType{
		Conn:    c,
		Dialect: d,
	}
	return completer.getCandidates
}
