package postgres

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/misc"
)

var postgresTypeNameToFormat = map[string][2]string{
	"TIMESTAMPTZ": [2]string{"TIMESTAMP WITH TIME ZONE", dialect.DateTimeTzLayout},
	"TIMESTAMP":   [2]string{"TIMESTAMP", dialect.DateTimeLayout},
	"DATE":        [2]string{"DATE", dialect.DateOnlyLayout},
	"TIMETZ":      [2]string{"TIME WITH TIME ZONE", dialect.TimeTzLayout},
	"TIME":        [2]string{"TIME", dialect.TimeOnlyLayout},
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
		with target as (
			select to_regclass($1)::oid as oid
		)
		select
			a.attname as "NAME",
			case a.attnotnull
				when true then 'NOT NULL'
				else 'NULL'
			end as "NULL?",
			format_type(a.atttypid, a.atttypmod) as "TYPE"
		from target t
		join pg_attribute a
		  on a.attrelid = t.oid
		where a.attnum > 0
		  and not a.attisdropped
		order by a.attnum`,
	SQLForTables: `
		select
			n.nspname || '.' || c.relname as full_name,
			n.nspname as table_schema,
			c.relname as table_name,
			case c.relkind
				when 'r' then 'BASE TABLE'
				when 'p' then 'PARTITIONED TABLE'
				when 'v' then 'VIEW'
				when 'm' then 'MATERIALIZED VIEW'
				when 'f' then 'FOREIGN TABLE'
			end as table_type,
			c.reltuples::bigint as estimated_rows,
			obj_description(c.oid,'pg_class') as remarks
		from pg_class c
		join pg_namespace n
		  on n.oid = c.relnamespace
		where c.relkind in ('r','p','v','m','f')
		order by
			case n.nspname
				when 'pg_catalog' then 9
				when 'information_schema' then 8
				else 0
			end,
			case c.relkind
				when 'r' then 0
				when 'p' then 1
				else 9
			end,
			n.nspname,
			c.relname`,
	TypeConverterFor:  postgresTypeNameToConv,
	PlaceHolder:       &placeHolder{},
	TableNameField:    "full_name",
	ColumnNameField:   "name",
	IsTransactionSafe: canUseInTransaction,
	FormatValue:       formatValue,
}

func formatValue(typeName string, value any) (string, bool) {
	t, ok := value.(time.Time)
	if !ok {
		return "", false
	}
	f, ok := postgresTypeNameToFormat[typeName]
	if !ok {
		return "", false
	}
	return t.Format(f[1]), true
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
