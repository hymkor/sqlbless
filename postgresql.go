package main

import (
	"fmt"

	_ "github.com/lib/pq"
)

var postgresTypeNameToFormat = map[string][2]string{
	"TIMESTAMPTZ": [2]string{"TIMESTAMP WITH TIME ZONE", dateTimeTzLayout},
	"TIMESTAMP":   [2]string{"TIMESTAMP", dateTimeLayout},
	"DATE":        [2]string{"DATE", dateOnlyLayout},
	"TIMETZ":      [2]string{"TIME WITH TIME ZONE", timeTzLayout},
	"TIME":        [2]string{"TIME", timeTzLayout},
}

func postgresTypeNameToConv(typeName string) func(string) (string, error) {
	if f, ok := postgresTypeNameToFormat[typeName]; ok {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s '%s'", f[0], dt.Format(f[1])), nil
		}
	}
	return nil
}

var postgresSpec = &DBSpec{
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
      select schemaname,tablename,tableowner
        from pg_tables`,
	DisplayDateTimeLayout: dateTimeTzLayout,
	TypeNameToConv:        postgresTypeNameToConv,
}

func init() {
	RegisterDB("POSTGRES", postgresSpec)
}
