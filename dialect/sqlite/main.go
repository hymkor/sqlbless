package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite/compat"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/misc"
)

//go:embed tables.sql
var tablesSql string

//go:embed columns.sql
var columnsSql string

var Entry = &dialect.Entry{
	Usage:             "sqlbless sqlite3 :memory: OR <FILEPATH>",
	SQLForTables:      tablesSql,
	TypeConverterFor:  typeNameToConv,
	PlaceHolder:       &placeHolder{},
	SQLForColumns:     columnsSql,
	TableNameField:    "name",
	ColumnNameField:   "name",
	IsTransactionSafe: canUseInTransaction,
	IsQuerySQL: func(s string) bool {
		s, _ = misc.CutField(s)
		return strings.EqualFold(s, "PRAGMA")
	},
	FormatValue: formatValue,
}

func formatValue(typeName string, value any) (string, bool) {
	if t, ok := value.(time.Time); ok {
		if spec, ok := typeNameToHolder[typeName]; ok {
			return t.Format(spec[1]), true
		}
	}
	return "", false
}

func canUseInTransaction(sql string) bool {
	keyword, _ := misc.CutField(sql)
	keyword = strings.TrimRight(keyword, ";")
	return !strings.EqualFold(keyword, "VACUUM")
}

var typeNameToHolder = map[string][2]string{
	"TIMESTAMP": [2]string{
		"strftime('%Y-%m-%d %H:%M:%f',?)",
		dialect.DateTimeLayout},
	"TIME": [2]string{
		"time(?)",
		dialect.TimeOnlyLayout0p},
	"DATE": [2]string{
		"date(?)",
		dialect.DateOnlyLayout},
	"DATETIME": [2]string{
		"datetime(?)",
		dialect.DateTimeLayout0p},
}

func typeNameToConv(typeName string) func(string) (any, error) {
	if holder, ok := typeNameToHolder[typeName]; ok {
		return func(s string) (any, error) {
			dt, err := dialect.ParseAnyDateTime(s)
			if err != nil {
				return s, nil
			}
			return &dialect.SQLFmtAndValue{
				Format: holder[0],
				Value:  dt.Format(holder[1]),
			}, nil
		}
	}
	return nil
}

type placeHolder struct {
	values []any
}

func (ph *placeHolder) Make(v any) string {
	if w, ok := v.(*dialect.SQLFmtAndValue); ok {
		ph.values = append(ph.values, w.Value)
		return strings.ReplaceAll(w.Format, "?", fmt.Sprintf("$v%d", len(ph.values)))
	}
	ph.values = append(ph.values, v)
	return fmt.Sprintf("$v%d", len(ph.values))
}

func (ph *placeHolder) NormalizeColumnForWhere(value any, columnName string) string {
	if w, ok := value.(*dialect.SQLFmtAndValue); ok {
		return strings.ReplaceAll(w.Format, "?", columnName)
	}
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
	Entry.Register("SQLITE3")
}
