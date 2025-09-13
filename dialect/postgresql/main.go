package sqlbless

import (
	"fmt"

	_ "github.com/lib/pq"

	"github.com/hymkor/sqlbless/dialect"
)

var postgresTypeNameToFormat = map[string][2]string{
	"TIMESTAMPTZ": [2]string{"TIMESTAMP WITH TIME ZONE", dialect.DateTimeTzLayout},
	"TIMESTAMP":   [2]string{"TIMESTAMP", dialect.DateTimeLayout},
	"DATE":        [2]string{"DATE", dialect.DateOnlyLayout},
	"TIMETZ":      [2]string{"TIME WITH TIME ZONE", dialect.TimeTzLayout},
	"TIME":        [2]string{"TIME", dialect.TimeTzLayout},
}

func postgresTypeNameToConv(typeName string) func(string) (string, error) {
	if f, ok := postgresTypeNameToFormat[typeName]; ok {
		return func(s string) (string, error) {
			dt, err := dialect.ParseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s '%s'", f[0], dt.Format(f[1])), nil
		}
	}
	return nil
}

var postgresSpec = &dialect.Entry{
	Usage: "sqlbless postgres://<USERNAME>:<PASSWORD>@<HOSTNAME>:<PORT>/<DBNAME>?sslmode=disable",
	SqlForDesc: `
      select a.attnum as "ID",
             a.attname as "NAME",
             case
               when t.typname = 'varchar' then 'varchar(' || ( a.atttypmod - 4 )  || ')'
               when a.atttypmod >= 0 then t.typname || '(' || a.atttypmod || ')'
               else t.typname
             end as "TYPE",
             case
               when a.attnotnull then 'NOT NULL'
               else 'NULL'
             end as "NULL?"
        from pg_attribute a, pg_class c, pg_type t
       where a.attrelid = c.oid
         and c.relname = $1
         and a.attnum > 0
         and t.oid = a.atttypid
         and a.attisdropped is false
       order by a.attnum`,
	SqlForTab: `
      select *
        from information_schema.tables
       where table_type = 'BASE TABLE'
         and table_schema not in ('pg_catalog', 'information_schema')`,
	DisplayDateTimeLayout: dialect.DateTimeTzLayout,
	TypeNameToConv:        postgresTypeNameToConv,
	TableField:            "table_name",
	ColumnField:           "name",
}

func init() {
	dialect.Register("POSTGRES", postgresSpec)
}
