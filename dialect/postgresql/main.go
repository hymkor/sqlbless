package postgres

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/misc"
)

//go:embed tables.sql
var tablesSql string

//go:embed columns.sql
var columnsSql string

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
	Usage:             "sqlbless postgres://<USERNAME>:<PASSWORD>@<HOSTNAME>:<PORT>/<DBNAME>?sslmode=disable",
	SQLForColumns:     columnsSql,
	SQLForTables:      tablesSql,
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
