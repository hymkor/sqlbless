package sqlbless

import (
	"strings"

	_ "github.com/sijms/go-ora/v2"

	"github.com/hymkor/sqlbless/dialect"
)

func oracleTypeNameToConv(typeName string) func(string) (any, error) {
	if strings.HasPrefix(typeName, "TIMESTAMP") || typeName == "DATE" {
		return func(s string) (any, error) {
			return dialect.ParseAnyDateTime(s)
		}
	}
	return nil
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
	PlaceHolder:           &dialect.PlaceHolderName{Prefix: ":", Format: "v"},
}

func init() {
	dialect.Register("ORACLE", oracleSpec)
}
