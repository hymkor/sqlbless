package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite/compat"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/misc"
)

var Entry = &dialect.Entry{
	Usage: "sqlbless sqlite3 :memory: OR <FILEPATH>",
	SQLForTables: `
	select 'main' as schema,name,rootpage,sql from sqlite_master
	where type = 'table'
	union all
	select 'temp' as schema,name,rootpage,sql from sqlite_temp_master
	where type = 'table'`,
	TypeConverterFor:  typeNameToConv,
	PlaceHolder:       &placeHolder{},
	SQLForColumns:     `PRAGMA table_info({table_name})`,
	TableNameField:    "name",
	ColumnNameField:   "name",
	IsTransactionSafe: canUseInTransaction,
	IsQuerySQL: func(s string) bool {
		s, _ = misc.CutField(s)
		return strings.EqualFold(s, "PRAGMA")
	},
	FormatValue: formatValue,
}

func formatValue(typeName string, value any) (string, bool) {
	if t, ok := value.(time.Time); ok {
		if typeName == "DATE" {
			return t.Format("2006-01-02"), true
		}
		if typeName == "DATETIME" {
			return t.Format("2006-01-02 15:04:05"), true
		}
		if typeName == "TIMESTAMP" {
			return t.Format("2006-01-02 15:04:05.999999999"), true
		}
	}
	return "", false
}

func canUseInTransaction(sql string) bool {
	keyword, _ := misc.CutField(sql)
	keyword = strings.TrimRight(keyword, ";")
	return !strings.EqualFold(keyword, "VACUUM")
}

var typeNameToHolder = map[string][2]string{
	"TIMESTAMP": [2]string{
		"strftime('%Y-%m-%d %H:%M:%f',?)", // "2006-01-02 15:04:05.999999999-07:00"
		"2006-01-02 15:04:05.999999999"},
	"TIME": [2]string{
		"time(?)", // dialect.TimeOnlyLayout
		"15:04:05"},
	"DATE": [2]string{
		"date(?)", // dialect.DateOnlyLayout
		"2006-01-02"},
	"DATETIME": [2]string{
		"datetime(?)", // dialect.DateTimeLayout
		"2006-01-02 15:04:05"},
}

func typeNameToConv(typeName string) func(string) (any, error) {
	if holder, ok := typeNameToHolder[typeName]; ok {
		return func(s string) (any, error) {
			dt, err := dialect.ParseAnyDateTime(s)
			if err != nil {
				return s, nil
			}
			return &withHolder{
				holder: holder[0],
				value:  dt.Format(holder[1]),
			}, nil
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
	Entry.Register("SQLITE3")
}
