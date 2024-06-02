package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var mySQLTypeNameToFormat = map[string]string{
	"DATETIME":  dateTimeFormat,
	"TIMESTAMP": dateTimeFormat,
	"TIME":      timeOnlyFormat,
	"DATE":      dateOnlyFormat,
}

func mySQLTypeNameToConv(typeName string) func(string) (string, error) {
	if format, ok := mySQLTypeNameToFormat[typeName]; ok {
		return func(s string) (string, error) {
			dt, err := parseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s '%s'", typeName, dt.Format(format)), nil
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
