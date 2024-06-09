package main

var sqliteSpec = &DBSpec{
	Usage: "sqlbless sqlite3 :memory: OR <FILEPATH>",
	SqlForTab: `
	select 'SCHEMA' AS SCHEMA,* from sqlite_master
	union all
	select 'TEMP_SCHEMA' AS SCHEMA,* FROM sqlite_temp_schema`,
	DisplayDateTimeLayout: dateTimeTzLayout,
	SqlForDesc:            `PRAGMA table_info({table_name})`,
}
