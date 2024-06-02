package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func mySQLTypeNameToConv(typeName string) func(string) (string, error) {
	switch typeName {
	case "DATETIME", "TIMESTAMP":
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("STR_TO_DATE('%s','%%Y-%%m-%%d %%H:%%i:%%s.%%f')", dt.Format(dateTimeFormat)), nil
		}
	case "TIME":
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("STR_TO_DATE('%s','%%H:%%i:%%s.%%f')", dt.Format(timeOnlyFormat)), nil
		}
	case "DATE":
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("STR_TO_DATE('%s','%%Y-%%m-%%d')", dt.Format(dateOnlyFormat)), nil
		}
	default:
		return nil
	}
}

var mySqlSpec = &DBSpec{
	Usage: `sqlbless mysql user:password@/dbname`,
	SqlForDesc: `
        select ordinal_position as "ID",
               column_name as "NAME",
               case
                 when character_maximum_length is not null then 
                      concat(data_type,'(',character_maximum_length,')')
                 when datetime_precision is not null then
                      concat(data_type,'(',datetime_precision,')')
                 else data_type
               end as "TYPE",
               case is_nullable
                 when "YES" then 'NULL'
                 else 'NOT NULL'
               end as "NULL?"
          from information_schema.columns
         where table_name = ?
         order by ordinal_position`,
	SqlForTab:      `select * from information_schema.tables`,
	TypeNameToConv: mySQLTypeNameToConv,
}
