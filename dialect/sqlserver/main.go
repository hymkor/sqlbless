package sqlserver

import (
	"time"

	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"

	"github.com/hymkor/sqlbless/dialect"
)

var typeSpec = map[string][2]string{
	"DATE": {
		"CAST(@v%d AS DATE)",
		dialect.DateOnlyLayout},
	"SMALLDATETIME": {
		"CAST(@v%d AS SMALLDATETIME)",
		dialect.ShortDateTimeLayout},
	"DATETIME": {
		"CAST(@v%d AS DATETIME)",
		dialect.DateTimeLayout3p},
	"DATETIME2": {
		"CAST(@v%d AS DATETIME2)",
		dialect.DateTimeLayout7p},
	"DATETIMEOFFSET": {
		"CAST(@v%d AS DATETIMEOFFSET)",
		dialect.DateTimeTzLayout},
	"TIME": {
		"CAST(@v%d AS TIME)",
		dialect.TimeOnlyLayout},
}

func typeNameToConv(typeName string) func(string) (any, error) {
	spec, ok := typeSpec[typeName]
	if !ok {
		return nil
	}
	return func(s string) (any, error) {
		dt, err := dialect.ParseAnyDateTime(s)
		if err != nil {
			return nil, err
		}
		return &dialect.SQLFmtAndValue{
			Format: spec[0],
			Value:  dt.Format(spec[1]),
		}, nil
	}
}

func formatValue(typeName string, value any) (string, bool) {
	t, ok := value.(time.Time)
	if !ok {
		return "", false
	}
	spec, ok := typeSpec[typeName]
	if !ok {
		return "", false
	}
	return t.Format(spec[1]), true
}

var sqlServerSpec = &dialect.Entry{
	Usage: "sqlbless sqlserver://@<HOSTNAME>?database=<DBNAME>",
	SQLForColumns: `
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
	SQLForTables:     `select * from sys.tables`,
	TypeConverterFor: typeNameToConv,
	PlaceHolder:      &dialect.PlaceHolderName{Mark: "@", Prefix: "v"},
	TableNameField:   "name",
	ColumnNameField:  "name",
	FormatValue:      formatValue,
}

func init() {
	sqlServerSpec.Register("SQLSERVER")
}
