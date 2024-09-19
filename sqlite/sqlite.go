package sqlite

import (
	_ "github.com/glebarez/go-sqlite/compat"

	"github.com/hymkor/sqlbless"
)

const dateTimeTzLayout = "2006-01-02 15:04:05.999999999 -07:00"

var Dialect = &sqlbless.DBDialect{
	Usage: "sqlbless sqlite3 :memory: OR <FILEPATH>",
	SqlForTab: `
	select 'SCHEMA' AS SCHEMA,* from sqlite_master
	union all
	select 'TEMP_SCHEMA' AS SCHEMA,* FROM sqlite_temp_schema`,
	DisplayDateTimeLayout: dateTimeTzLayout,
	SqlForDesc:            `PRAGMA table_info({table_name})`,
}

func init() {
	sqlbless.RegisterDB("SQLITE3", Dialect)
}
