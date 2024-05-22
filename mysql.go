package main

import (
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func mySQLTypeNameToConv(typeName string) func(string) (string, error) {
	if strings.Contains(typeName, "DATETIME") {
		return func(s string) (string, error) {
			_, err := time.Parse(dateTimeFormat, s)
			if err != nil {
				return "", err
			}
			return "STR_TO_DATE('" + s + "','%Y-%m-%d %H:%i:%s')", nil
		}
	}
	if strings.Contains(typeName, "TIME") {
		return func(s string) (string, error) {
			_, err := time.Parse(dateTimeFormat, s)
			if err != nil {
				return "", err
			}
			return "STR_TO_DATE('" + s + "','%H:%i:%s')", nil
		}
	}
	if strings.Contains(typeName, "DATE") {
		return func(s string) (string, error) {
			_, err := time.Parse(dateOnlyFormat, s)
			if err != nil {
				return "", err
			}
			return "STR_TO_DATE('" + s + "','%Y-%m-%d')", nil
		}
	}
	return nil
}

var mySqlSpec = &DBSpec{
	Usage: `sqlbless mysql user:password@/dbname`,
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
	SqlForTab:      `select * from information_schema.tables`,
	TypeNameToConv: mySQLTypeNameToConv,
}
