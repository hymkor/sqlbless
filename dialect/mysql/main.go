package sqlbless

import (
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hymkor/sqlbless/dialect"
)

const (
	mySQLDateTimeTzLayout = "2006-01-02 15:04:05.999999999-07:00"
)

var mySQLTypeNameToFormat = map[string]string{
	"DATETIME":  dialect.DateTimeLayout,
	"TIMESTAMP": dialect.DateTimeLayout,
	"TIME":      dialect.TimeOnlyLayout,
	"DATE":      dialect.DateOnlyLayout,
}

func typeNameToConv(typeName string) func(string) (any, error) {
	f, ok := mySQLTypeNameToFormat[typeName]
	if !ok {
		return nil
	}
	return func(s string) (any, error) {
		typ, t, err := dialect.ParseAnyDateTimeX(s)
		if err != nil {
			return nil, err
		}
		if typ == dialect.DateTimeTzLayout {
			// for parseTime=true
			return t.Format(mySQLDateTimeTzLayout), nil
		}
		return t.Format(f), nil
	}
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
