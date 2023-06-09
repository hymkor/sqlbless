package main

import (
	"fmt"
	"io"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

type DBSpec struct {
	SqlForDesc string
	SqlForTab  string
}

var dbSpecs = map[string]*DBSpec{
	"POSTGRES": &DBSpec{
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
	},
	"ORACLE": &DBSpec{
		SqlForDesc: `
      select column_id as "ID",
             column_name as "NAME",
             case data_type
               when 'NUMBER' then data_type
               when 'DATE' then data_type
               else data_type || '(' || data_length || ')'
             end as "TYPE",
             case
               when nullable = 'Y' THEN 'NULL'
               else 'NOT NULL'
             end as "NULL?"
        from all_tab_columns
       where table_name = UPPER(:1)
       order by column_id`,
		SqlForTab: `select * from tab`,
	},
	"SQLSERVER": &DBSpec{
		SqlForDesc: `
        select c.column_id as "ID",
               c.name as "NAME",
               case
                 when c.max_length > 0 then
                   t.name + '(' + convert(varchar,c.max_length) + ')'
                 else
                   t.name
               end as "TYPE",
               case c.is_nullable
                 when 1 then 'NULL'
                 else 'NOT NULL'
               end as "NULL?"
          from sys.columns c,
               sys.objects o,
               sys.types t
         where c.object_id = o.object_id
           and o.name = @p1
           and c.user_type_id = t.user_type_id
         order by c.column_id`,
		SqlForTab: `select * from sys.objects`,
	},
	"MYSQL": &DBSpec{
		SqlForDesc: `
        select ordinal_position as "ID",
               column_name as "NAME",
               case
                 when character_maximum_length is null then data_type
                 else concat(data_type,'(',character_maximum_length,')')
               end as "TYPE",
               case is_nullable
                 when "YES" then 'NULL'
                 else 'NOT NULL'
               end as "NULL?"
          from information_schema.columns
         where table_name = ?
         order by ordinal_position`,
		SqlForTab: `select * from information_schema.tables`,
	},
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, `  sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE`)
	fmt.Fprintln(w, `  sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"`)
	fmt.Fprintln(w, `  sqlbless sqlserver "sqlserver://@localhost?database=master"`)
}
