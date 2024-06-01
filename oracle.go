package main

import (
	"fmt"
	"strings"

	_ "github.com/sijms/go-ora/v2"
)

func oracleTypeNameToConv(typeName string) func(string) (string, error) {
	if strings.Contains(typeName, "TIMESTAMP") {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("TO_TIMESTAMP('%s','YYYY-MM-DD HH24:MI:SS.FF')", dt.Format(dateTimeFormat)), nil
		}
	}
	if strings.Contains(typeName, "DATE") {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("TO_DATE('%s','YYYY-MM-DD HH24:MI:SS')", dt.Format(dateTimeFormat)), nil
		}
	}
	return nil
}

var oracleSpec = &DBSpec{
	Usage: "sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE",
	SqlForDesc: `
  select column_id as "ID",
		 column_name as "NAME",
		 case 
		   when data_type = 'NUMBER' then data_type
		   when data_type = 'DATE' then data_type
		   when data_type like 'TIMESTAMP%' then data_type
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
