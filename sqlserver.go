package main

import (
	"fmt"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

func sqlServerTypeNameToConv(typeName string) func(string) (string, error) {
	switch typeName {
	case "SMALLDATETIME":
		// SMALLDATETIME: YYYY-MM-DD hh:mm:ss
		// 120: yyyy-mm-dd hh:mi:ss
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("CONVERT(SMALLDATETIME,'%s',120)", dt.Format("2006-01-02 15:04:05")), nil
		}
	case "DATETIME", "DATETIME2":
		// datetime      for YYYY-MM-DD hh:mm:ss[.nnn]
		// datetime2     for YYYY-MM-DD hh:mm:ss[.nnnnnnn]
		// 121: yyyy-mm-dd hh:mi:ss.mmm
		// 120: yyyy-mm-dd hh:mi:ss
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			if dt.Nanosecond() > 0 {
				return fmt.Sprintf("CONVERT(%s,'%s',121)", typeName, dt.Format(dateTimeLayout)), nil
			}
			return fmt.Sprintf("CONVERT(%s,'%s',120)", typeName, dt.Format("2006-01-02 15:04:05")), nil
		}
	case "DATE":
		// DATE: YYYY-MM-DD
		// 23: yyyy-mm-dd
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("CONVERT(DATE,'%s',23)", dt.Format(dateOnlyLayout)), nil
		}
	case "TIME":
		// TIME: hh:mm:ss[.nnnnnnn]
		// 114: hh:mi:ss:mmm !!!COLON!!!
		// 108: hh:mi:ss
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			if dt.Nanosecond() > 0 {
				return fmt.Sprintf("CONVERT(TIME,'%s',114)", strings.Replace(dt.Format(timeOnlyLayout), ".", ":", 1)), nil
				//return fmt.Sprintf("CONVERT(%s,'%s',121)", typeName, dt.Format(dateTimeFormat)), nil
			}
			return fmt.Sprintf("CONVERT(TIME,'%s',108)", dt.Format("15:04:05")), nil
		}
	case "DATETIMEOFFSET":
		// DATETIMEOFFSET
		// 127: yyyy-MM-ddThh:mm:ss.fffZ (スペースなし)
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("CONVERT(DATETIMEOFFSET,'%s',127)", dt.Format(dateTimeTzLayout)), nil
		}
	default:
		return nil
	}
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
	SqlForTab:             `select * from sys.objects`,
	DisplayDateTimeLayout: dateTimeTzLayout,
	TypeNameToConv:        sqlServerTypeNameToConv,
}
