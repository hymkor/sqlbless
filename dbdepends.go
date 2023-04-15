package main

import (
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
)

type Options struct {
	DontRollbackOnFail bool
	SqlForDesc         string
}

var dbDependent = map[string]*Options{
	"POSTGRES": &Options{
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
	},
	"ORACLE": &Options{
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
	},
}
