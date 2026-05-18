package sqlbless

import (
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"

	"github.com/hymkor/sqlbless/dialect"
)

var oracleSpec = &dialect.Entry{
	Usage: "sqlbless oracle://<USERNAME>:<PASSWORD>@<HOSTNAME>:<PORT>/<SERVICE>",
	SQLForColumns: `
  select column_id as "ID",
		 column_name as "NAME",
		 case 
		   when data_type = 'NUMBER' then data_type
		   when data_type = 'DATE' then data_type
		   when data_type like 'TIMESTAMP%' then data_type
		   else data_type || '(' || data_length || ')'
		 end as "TYPE",
		 case
		   when nullable = 'Y' THEN 'NULL'
		   else 'NOT NULL'
		 end as "NULL?"
	from all_tab_columns
   where table_name = UPPER(:1)
   order by column_id`,
	SQLForTables:     `select * from tab where tname not like 'BIN$%'`,
	TypeConverterFor: typeNameToConv,
	TableNameField:   "tname",
	ColumnNameField:  "name",
	PlaceHolder:      &dialect.PlaceHolderName{Mark: ":", Prefix: "v"},
	FormatValue:      formatValue,
}

const (
	timeStampTz  = "TimeStampTZ_DTY"
	timeStampLtz = "TimeStampLTZ_DTY"
)

func formatValue(typeName string, value any) (string, bool) {
	t, ok := value.(time.Time)
	if !ok {
		return "", false
	}
	if typeName == "DATE" {
		return t.Format(dialect.DateTimeLayout0p), true
	}
	if strings.EqualFold(typeName, timeStampTz) ||
		strings.EqualFold(typeName, timeStampLtz) {

		return t.Format("2006-01-02 15:04:05.999999 -07:00"), true
	}
	return t.Format("2006-01-02 15:04:05.999999"), true
}

func typeNameToConv(typeName string) func(string) (any, error) {
	var format string
	var layout string

	if typeName == "DATE" {
		format = "TO_DATE(:v%d,'YYYY-MM-DD HH24:MI:SS')"
		layout = dialect.DateTimeLayout0p
	} else if strings.HasPrefix(typeName, "TIMESTAMP") {
		if strings.EqualFold(typeName, timeStampTz) ||
			strings.EqualFold(typeName, timeStampLtz) {

			format = "TO_TIMESTAMP_TZ(:v%d,'YYYY-MM-DD HH24:MI:SS.FF TZH:TZM')"
			layout = "2006-01-02 15:04:05.999999 -07:00"
		} else {
			format = "TO_TIMESTAMP(:v%d,'YYYY-MM-DD HH24:MI:SS.FF')"
			layout = "2006-01-02 15:04:05.999999"
		}
	} else {
		return nil
	}
	return func(s string) (any, error) {
		dt, err := dialect.ParseAnyDateTime(s)
		if err != nil {
			return s, nil
		}
		return &dialect.SQLFmtAndValue{
			Format: format,
			Value:  dt.Format(layout),
		}, nil
	}
}

func init() {
	oracleSpec.Register("ORACLE")
}
