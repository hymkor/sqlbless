package postgres

import (
	"fmt"
	"strings"

	_ "github.com/lib/pq"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/misc"
)

var postgresTypeNameToFormat = map[string][2]string{
	"TIMESTAMPTZ": [2]string{"TIMESTAMP WITH TIME ZONE", dialect.DateTimeTzLayout},
	"TIMESTAMP":   [2]string{"TIMESTAMP", dialect.DateTimeLayout},
	"DATE":        [2]string{"DATE", dialect.DateOnlyLayout},
	"TIMETZ":      [2]string{"TIME WITH TIME ZONE", dialect.TimeTzLayout},
	"TIME":        [2]string{"TIME", dialect.TimeTzLayout},
}

func postgresTypeNameToConv(typeName string) func(string) (any, error) {
	if _, ok := postgresTypeNameToFormat[typeName]; ok {
		return func(s string) (any, error) {
			return dialect.ParseAnyDateTime(s)
		}
	}
	return nil
}

type placeHolder struct {
	values []any
}

func (ph *placeHolder) Make(v any) string {
	ph.values = append(ph.values, v)
	return fmt.Sprintf("$%d", len(ph.values))
}

func (ph *placeHolder) Values() (result []any) {
	result = ph.values
	ph.values = ph.values[:0]
	return
}

var postgresSpec = &dialect.Entry{
	Usage: "sqlbless postgres://<USERNAME>:<PASSWORD>@<HOSTNAME>:<PORT>/<DBNAME>?sslmode=disable",
	SQLForColumns: `
      select a.attnum as "ID",
             a.attname as "NAME",
             case
               when t.typname = 'varchar' then 'varchar(' || ( a.atttypmod - 4 )  || ')'
               when a.atttypmod >= 0 then t.typname || '(' || a.atttypmod || ')'
               else t.typname
             end as "TYPE",
             case
               when a.attnotnull then 'NOT NULL'
               else 'NULL'
             end as "NULL?"
        from pg_attribute a, pg_class c, pg_type t
       where a.attrelid = c.oid
         and c.relname = $1
         and a.attnum > 0
         and t.oid = a.atttypid
         and a.attisdropped is false
       order by a.attnum`,
	SQLForTables: `
      select *
        from information_schema.tables
       where table_type = 'BASE TABLE'
         and table_schema not in ('pg_catalog', 'information_schema')`,
	TypeConverterFor:    postgresTypeNameToConv,
	PlaceHolder:         &placeHolder{},
	TableNameField:      "table_name",
	ColumnNameField:     "name",
	CanUseInTransaction: canUseInTransaction,
}

func canUseInTransaction(sql string) bool {
	keyword, rest := misc.CutField(sql)
	keyword = strings.TrimRight(keyword, ";")
	switch strings.ToUpper(keyword) {
	case "VACUUM", "REINDEX", "CLUSTER":
		return false
	case "CREATE", "DROP":
		keyword, _ = misc.CutField(rest)
		return !strings.EqualFold(keyword, "DATABASE") && !strings.EqualFold(keyword, "TABLESPACE")
	default:
		return true
	}
}

func init() {
	postgresSpec.Register("POSTGRES")
}
