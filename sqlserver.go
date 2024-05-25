package main

import (
	"fmt"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

func sqlServerTypeNameToConv(typeName string) func(string) (string, error) {
	if strings.Contains(typeName, "DATETIME") {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("CONVERT(DATETIME,'%s',120)", dt.Format(dateTimeFormat)), nil
		}
	}
	if strings.Contains(typeName, "DATE") {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("CONVERT(DATE,'%s',23)", dt.Format(dateOnlyFormat)), nil
		}
	}
	if strings.Contains(typeName, "TIME") {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("CONVERT(TIME,'%s',108)", dt.Format(timeOnlyFormat)), nil
		}
	}
	return nil
}

var sqlServerSpec = &DBSpec{
	Usage: `sqlbless sqlserver "sqlserver://@localhost?database=master"`,
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
	SqlForTab:      `select * from sys.objects`,
	TypeNameToConv: sqlServerTypeNameToConv,
}
