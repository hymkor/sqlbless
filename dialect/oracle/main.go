package sqlbless

import (
	"database/sql"
	"fmt"
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
	TypeConverterFor: oracleTypeNameToConv,
	TableNameField:   "tname",
	ColumnNameField:  "name",
	PlaceHolder:      new(placeHolder),
	FormatValue:      formatValue,
}

func formatValue(typeName string, value any) (string, bool) {
	t, ok := value.(time.Time)
	if !ok {
		return "", false
	}
	if typeName == "DATE" {
		return t.Format("2006-01-02 15:04:05"), true
	}
	if strings.EqualFold(typeName, "TimeStampTZ_DTY") ||
		strings.EqualFold(typeName, "TimeStampLTZ_DTY") {

		return t.Format("2006-01-02 15:04:05.999999 -07:00"), true
	}
	return t.Format("2006-01-02 15:04:05.999999"), true
}

type withFormat struct {
	format string
	value  any
}

func oracleTypeNameToConv(typeName string) func(string) (any, error) {
	var format string
	var layout string

	if typeName == "DATE" {
		format = "TO_DATE(:v%d,'YYYY-MM-DD HH24:MI:SS')"
		layout = "2006-01-02 15:04:05"
	} else if strings.HasPrefix(typeName, "TIMESTAMP") {
		if strings.EqualFold(typeName, "TimeStampTZ_DTY") ||
			strings.EqualFold(typeName, "TimeStampLTZ_DTY") {

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
		return &withFormat{
			format: format,
			value:  dt.Format(layout),
		}, nil
	}
}

type placeHolder struct {
	values []any
}

func (ph *placeHolder) Make(v any) string {
	if w, ok := v.(*withFormat); ok {
		ph.values = append(ph.values, w.value)
		return fmt.Sprintf(w.format, len(ph.values))
	}
	ph.values = append(ph.values, v)
	return fmt.Sprintf(":v%d", len(ph.values))
}

func (ph *placeHolder) NormalizeColumnForWhere(value any, columnName string) string {
	return columnName
}

func (ph *placeHolder) Values() (result []any) {
	for i, v := range ph.values {
		result = append(result, sql.Named(fmt.Sprintf("v%d", i+1), v))
	}
	ph.values = ph.values[:0]
	return
}

func init() {
	oracleSpec.Register("ORACLE")
}
