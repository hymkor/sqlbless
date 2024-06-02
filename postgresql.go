package main

import (
	"fmt"

	_ "github.com/lib/pq"
)

var postgresTypeNameToFormat = map[string]string{
	"TIMESTAMP": dateTimeFormat,
	"DATE":      dateOnlyFormat,
	"TIME":      timeOnlyFormat,
}

func postgresTypeNameToConv(typeName string) func(string) (string, error) {
	if format, ok := postgresTypeNameToFormat[typeName]; ok {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", fmt.Errorf("postgresql.go: %w", err)
			}
			return fmt.Sprintf("%s '%s'", typeName, dt.Format(format)), nil
		}
	}
	return nil
}

var postgresSpec = &DBSpec{
	Usage: `sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"`,
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
	TypeNameToConv: postgresTypeNameToConv,
}
