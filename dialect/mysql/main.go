package sqlbless

import (
	_ "embed"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hymkor/sqlbless/dialect"
)

const (
	mySQLDateTimeTzLayout = "2006-01-02 15:04:05.999999999-07:00"
)

//go:embed tables.sql
var tablesSql string

//go:embed columns.sql
var columnsSql string

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
	Usage:            `sqlbless mysql <USERNAME>:<PASSWORD>@/<DBNAME>`,
	SQLForColumns:    columnsSql,
	SQLForTables:     tablesSql,
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
