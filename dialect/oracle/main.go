package sqlbless

import (
	"fmt"
	"strings"

	_ "github.com/sijms/go-ora/v2"

	"github.com/hymkor/sqlbless/dialect"
)

func oracleTypeNameToConv(typeName string) func(string) (string, error) {
	var sfmt string
	var dfmt string
	if strings.HasPrefix(typeName, "TIMESTAMP") {
		// sfmt = "TO_TIMESTAMP('%s','YYYY-MM-DD HH24:MI:SS.FF')"
		sfmt = "TO_TIMESTAMP_TZ('%s','YYYY-MM-DD HH24:MI:SS.FF TZH:TZM')"
		dfmt = dialect.DateTimeTzLayout
	} else if typeName == "DATE" {
		sfmt = "TO_DATE('%s','YYYY-MM-DD HH24:MI:SS')"
		dfmt = dialect.DateTimeLayout
	} else {
		return nil
	}
	return func(s string) (string, error) {
		dt, err := dialect.ParseAnyDateTime(s)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(sfmt, dt.Format(dfmt)), nil
	}
}

var oracleSpec = &dialect.Entry{
	Usage: "sqlbless oracle://<USERNAME>:<PASSWORD>@<HOSTNAME>:<PORT>/<SERVICE>",
	SqlForDesc: `
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
	SqlForTab:             `select * from tab where tname not like 'BIN$%'`,
	DisplayDateTimeLayout: dialect.DateTimeTzLayout,
	TypeNameToConv:        oracleTypeNameToConv,
	TableField:            "tname",
	ColumnField:           "name",
}

func init() {
	dialect.Register("ORACLE", oracleSpec)
}
