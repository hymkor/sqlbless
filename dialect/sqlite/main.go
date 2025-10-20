package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/glebarez/go-sqlite/compat"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/misc"
)

const dateTimeTzLayout = "2006-01-02 15:04:05.999999999 -07:00"

var Entry = &dialect.Entry{
	Usage: "sqlbless sqlite3 :memory: OR <FILEPATH>",
	SqlForTab: `
	select      'master' AS schema,name,rootpage,sql FROM sqlite_master
	where type = 'table'
	union all
	select 'temp_schema' AS schema,name,rootpage,sql FROM sqlite_temp_schema
	where type = 'temp_schema'`,
	DisplayDateTimeLayout: dateTimeTzLayout,
	TypeNameToConv:        typeNameToConv,
	PlaceHolder:           &placeHolder{},
	SqlForDesc:            `PRAGMA table_info({table_name})`,
	TableField:            "name",
	ColumnField:           "name",
	CanUseInTransaction:   canUseInTransaction,
}

func canUseInTransaction(sql string) bool {
	keyword, _ := misc.CutField(sql)
	keyword = strings.TrimRight(keyword, ";")
	return !strings.EqualFold(keyword, "VACUUM")
}

var typeNameToHolder = map[string]string{
	"TIMESTAMP": "datetime(?)", // "2006-01-02 15:04:05.999999999-07:00"
	"TIME":      "time(?)",     // dialect.TimeOnlyLayout
	"DATE":      "date(?)",     // dialect.DateOnlyLayout
	"DATETIME":  "datetime(?)", // dialect.DateTimeLayout
}

func typeNameToConv(typeName string) func(string) (any, error) {
	if holder, ok := typeNameToHolder[typeName]; ok {
		return func(s string) (any, error) {
			dt, err := dialect.ParseAnyDateTime(s)
			if err != nil {
				return s, nil
			}
			return &withHolder{holder: holder, value: dt}, nil
		}
	}
	return nil
}

type withHolder struct {
	holder string
	value  any
}

type placeHolder struct {
	values []any
}

func (ph *placeHolder) Make(v any) string {
	if w, ok := v.(*withHolder); ok {
		ph.values = append(ph.values, w.value)
		return strings.ReplaceAll(w.holder, "?", fmt.Sprintf("$v%d", len(ph.values)))
	}
	ph.values = append(ph.values, v)
	return fmt.Sprintf("$v%d", len(ph.values))
}

func (ph *placeHolder) NormalizeColumnForWhere(value any, columnName string) string {
	if w, ok := value.(*withHolder); ok {
		return strings.ReplaceAll(w.holder, "?", columnName)
	}
	return columnName
}

func (ph *placeHolder) Values() (result []any) {
	for i, v := range ph.values {
		result = append(result, sql.Named(fmt.Sprintf("v%d", i+1), v))
	}
	ph.values = ph.values[:0]
	return
}

func init() {
	dialect.Register("SQLITE3", Entry)
}
