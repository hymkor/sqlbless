package sqlserver

import (
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"

	"github.com/hymkor/sqlbless/dialect"
)

func sqlServerTypeNameToConv(typeName string) func(string) (any, error) {
	switch typeName {
	case "SMALLDATETIME", "DATETIME", "DATETIME2":
		// SMALLDATETIME: YYYY-MM-DD hh:mm:ss
		// 120: yyyy-mm-dd hh:mi:ss
		return func(s string) (any, error) {
			return dialect.ParseAnyDateTime(s)
		}
	}
	return nil
}

var sqlServerSpec = &dialect.Entry{
	Usage: "sqlbless sqlserver://@<HOSTNAME>?database=<DBNAME>",
	SqlForDesc: `
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
	SqlForTab:      `select * from sys.tables`,
	TypeNameToConv: sqlServerTypeNameToConv,
	PlaceHolder:    &dialect.PlaceHolderName{Prefix: "@", Format: "v"},
	TableField:     "name",
	ColumnField:    "name",
}

func init() {
	dialect.Register("SQLSERVER", sqlServerSpec)
}
