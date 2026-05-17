package dialect

import (
	"database/sql"
	"fmt"
)

type SQLFmtAndValue struct {
	Format string
	Value  any
}

type PlaceHolderName struct {
	Mark   string // e.g., `@` or ``
	Prefix string // e.g., `v`
	values []any
}

func (ph *PlaceHolderName) Make(v any) string {
	if wf, ok := v.(*SQLFmtAndValue); ok {
		ph.values = append(ph.values, wf.Value)
		return fmt.Sprintf(wf.Format, len(ph.values))
	}
	ph.values = append(ph.values, v)
	return fmt.Sprintf("%s%s%d", ph.Mark, ph.Prefix, len(ph.values))
}

func (ph *PlaceHolderName) Values() (result []any) {
	for i, v := range ph.values {
		result = append(result, sql.Named(fmt.Sprintf("%s%d", ph.Prefix, i+1), v))
	}
	ph.values = ph.values[:0]
	return
}
