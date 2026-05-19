package sqlbless

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hymkor/sqlbless/dialect"
)

var mySQLTypeNameToFormat = map[string]string{
	"DATETIME":  dialect.DateTimeLayout,
	"TIMESTAMP": "2006-01-02 15:04:05.999999999-07:00", // no space before tz
	"TIME":      dialect.TimeOnlyLayout,
	"DATE":      dialect.DateOnlyLayout,
}

func typeNameToConv(typeName string) func(string) (any, error) {
	f, ok := mySQLTypeNameToFormat[typeName]
	if !ok {
		return nil
	}
	return func(s string) (any, error) {
		t, err := dialect.ParseAnyDateTime(s)
		if err != nil {
			return nil, err
		}
		return t.Format(f), nil
	}
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

func formatValue(typeName string, value any) (string, bool) {
	t, ok := value.(time.Time)
	if !ok {
		return "", false
	}
	switch typeName {
	case "DATE":
		return t.Format(dialect.DateOnlyLayout), true
	case "TIME":
		return t.Format(dialect.TimeOnlyLayout), true
	case "DATETIME", "TIMESTAMP":
		return t.Format(dialect.DateTimeLayout), true
	default:
		return "", false
	}
}

var mySqlSpec = &dialect.Entry{
	Usage: `sqlbless mysql <USERNAME>:<PASSWORD>@/<DBNAME>`,
	SQLForColumns: `
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
          join (select ? as x) v
         where table_name   = REGEXP_REPLACE(v.x,'^[^\\.]*\\.','')
           and table_schema =
               case
                 when instr(v.x,'.') >= 1 then
                      regexp_replace(v.x,'\\.[^\\.]*$','')
                 else database()
               end
         order by ordinal_position`,
	SQLForTables: `
        select concat(table_schema,'.',table_name) as FULL_NAME,
               tables.* from information_schema.tables
         where table_type = 'BASE TABLE'
           and table_schema 
        not in ('mysql', 'information_schema', 'performance_schema', 'sys')`,
	TypeConverterFor: typeNameToConv,
	PlaceHolder:      &dialect.PlaceHolderQuestion{},
	DSNFilter:        mySQLDSNFilter,
	TableNameField:   "FULL_NAME",
	ColumnNameField:  "NAME",

	IdentifierEncloser: func(s string) string {
		return "`" + strings.ReplaceAll(s, ".", "`.`") + "`"
	},
	FormatValue: formatValue,
}

func init() {
	mySqlSpec.Register("MYSQL")
}
