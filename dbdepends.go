package main

import (
	"fmt"
	"io"

	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

type DBSpec struct {
	DontRollbackOnFail bool
	SqlForDesc         string
	SqlForTab          string
}

var dbSpecs = map[string]*DBSpec{
	"POSTGRES": &DBSpec{
		DontRollbackOnFail: true,
		SqlForDesc: `
      select a.attnum as "ID", a.attname as "NAME",
              case
                when a.attnotnull then 'NOT NULL'
                else 'NULL'
              end as "NULL?",
              case
                when t.typname = 'varchar' then 'varchar(' || ( a.atttypmod - 4 )  || ')'
                when a.atttypmod >= 0 then t.typname || '(' || a.atttypmod || ')'
                else t.typname
              end as "TYPE"
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
	},
	"ORACLE": &DBSpec{
		DontRollbackOnFail: false,
		SqlForDesc: `
      select column_id as "ID", column_name as "NAME",
              case
                when nullable = 'Y' THEN 'NULL'
                else 'NOT NULL'
              end as "NULL?",
              case data_type
                when 'NUMBER' then data_type
                when 'DATE' then data_type
                else data_type || '(' || data_length || ')'
              end as "TYPE"
        from all_tab_columns
        where table_name = UPPER(:1)
        order by column_id`,
		SqlForTab: `select * from tab`,
	},
	"SQLSERVER": &DBSpec{
		DontRollbackOnFail: true,
		SqlForDesc: `
        SELECT c.column_id as "ID",
               c.name as "NAME",
               case c.is_nullable
                 when 1 then 'NULL'
                 else 'NOT NULL'
                end as "NULL?",
                case
                  when c.max_length > 0 then
                    t.name + '(' + convert(varchar,c.max_length) + ')'
                  else
                   t.name
                end as "TYPE"
          FROM sys.columns c,
               sys.objects o,
               sys.types t
        WHERE  c.object_id = o.object_id
          AND  o.name = @p1
          AND  c.user_type_id = t.user_type_id
         order by c.column_id`,
		SqlForTab: `select * from sys.objects`,
	},
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, `  sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE`)
	fmt.Fprintln(w, `  sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"`)
}
