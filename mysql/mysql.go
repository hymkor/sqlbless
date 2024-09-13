package sqlbless

import (
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hymkor/sqlbless"
)

var mySQLTypeNameToFormat = map[string]string{
	"DATETIME":  sqlbless.DateTimeLayout,
	"TIMESTAMP": "2006-01-02 15:04:05.999999999-07:00",
	"TIME":      sqlbless.TimeOnlyLayout,
	"DATE":      sqlbless.DateOnlyLayout,
}

func mySQLTypeNameToConv(typeName string) func(string) (string, error) {
	if format, ok := mySQLTypeNameToFormat[typeName]; ok {
		return func(s string) (string, error) {
			dt, err := sqlbless.ParseAnyDateTime(s)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s '%s'", typeName, dt.Format(format)), nil
		}
	}
	return nil
}

func mySQLDSNFilter(dsn string) (string, error) {
	base, param, ok := strings.Cut(dsn, "?")
	hash := make(map[string][]string)
	if ok {
		for _, pair := range strings.Split(param, "&") {
			left, right, ok := strings.Cut(pair, "=")
			if ok {
				hash[left] = append(hash[left], right)
			}
		}
	}
	if _, ok := hash["parseTime"]; !ok {
		hash["parseTime"] = []string{"true"}
	}
	if _, ok := hash["loc"]; !ok {
		hash["loc"] = []string{"Local"}
	}
	var newdsn strings.Builder
	newdsn.WriteString(base)
	delimiter := '?'
	for key, values := range hash {
		for _, v := range values {
			fmt.Fprintf(&newdsn, "%c%s=%s", delimiter, key, v)
			delimiter = '&'
		}
	}
	return newdsn.String(), nil
}

var mySqlSpec = &sqlbless.DBSpec{
	Usage: `sqlbless mysql <USERNAME>:<PASSWORD>@/<DBNAME>`,
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
	SqlForTab:             `select * from information_schema.tables`,
	DisplayDateTimeLayout: sqlbless.DateTimeTzLayout,
	TypeNameToConv:        mySQLTypeNameToConv,
	DSNFilter:             mySQLDSNFilter,
}

func init() {
	sqlbless.RegisterDB("MYSQL", mySqlSpec)
}
