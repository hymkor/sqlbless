package sqlite

import (
	"fmt"

	_ "github.com/glebarez/go-sqlite/compat"

	"github.com/hymkor/sqlbless/dialect"
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
	SqlForDesc:            `PRAGMA table_info({table_name})`,
	TableField:            "name",
	ColumnField:           "name",
}

var typeNameToFormat = map[string]string{
	"TIMESTAMP": "2006-01-02 15:04:05.999999999-07:00",
	"TIME":      dialect.TimeOnlyLayout,
	"DATE":      dialect.DateOnlyLayout,
}

func typeNameToConv(typeName string) func(string) (string, error) {
	if format, ok := typeNameToFormat[typeName]; ok {
		return func(s string) (string, error) {
			dt, err := dialect.ParseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("'%s'", dt.Format(format)), nil
		}
	}
	return nil
}

func init() {
	dialect.Register("SQLITE3", Entry)
}
