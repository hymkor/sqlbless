package main

import (
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

func oracleTypeNameToConv(typeName string) func(string) (string, error) {
	if !strings.Contains(typeName, "DATE") {
		return nil
	}
	return func(s string) (string, error) {
		_, err := time.Parse(dateTimeFormat, s)
		if err != nil {
			return "", err
		}
		return "TO_DATE('" + s + "','YYYY-MM-DD HH24:MI:SS')", nil
	}
}

var oracleSpec = &DBSpec{
	Usage: "sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE",
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
	SqlForTab:      `select * from tab`,
	TypeNameToConv: oracleTypeNameToConv,
}
