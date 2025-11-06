package dialect

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"database/sql"
)

var (
	ErrNotTimeFormat           = errors.New("not time format")
	ErrTooFewArguments         = errors.New("too few arguments")
	ErrDSNStringIsNotSpecified = errors.New("DSN String is not specified")
	ErrSupportDriveNotFound    = errors.New("support driver not found")
)

type PlaceHolder interface {
	Make(any) string
	Values() []any
}

type Entry struct {
	// Usage describes how to use this entry or what it represents.
	Usage string

	// SQLForColumns is the SQL query used to retrieve column information.
	SQLForColumns string

	// SQLForTables is the SQL query used to retrieve table information.
	SQLForTables string

	// TableNameField is the field name for table names in SQL results.
	TableNameField string

	// ColumnNameField is the field name for column names in SQL results.
	ColumnNameField string

	// PlaceHolder defines how to represent placeholders (e.g., ?, $1) in SQL.
	PlaceHolder PlaceHolder

	// TypeConverterFor returns a converter function for a given type name.
	// The returned function converts a string literal to the corresponding Go value.
	TypeConverterFor func(typeName string) func(literal string) (any, error)

	// DSNFilter adjusts or validates a given DSN string before use.
	DSNFilter func(dsn string) (string, error)

	// IsTransactionSafe reports whether the given SQL statement is safe to run in a transaction.
	IsTransactionSafe func(sql string) bool

	// IsQuerySQL reports whether the given SQL statement is a query (e.g., SELECT) or not.
	IsQuerySQL func(sql string) bool
}

func (D *Entry) LookupConverter(typeName string) func(string) (any, error) {
	if D.TypeConverterFor == nil {
		return nil
	}
	return D.TypeConverterFor(typeName)
}

const (
	DateTimeTzLayout = "2006-01-02 15:04:05.999999999 -07:00"
	DateTimeLayout   = "2006-01-02 15:04:05.999999999"
	DateOnlyLayout   = "2006-01-02"
	TimeOnlyLayout   = "15:04:05.999999999"
	TimeTzLayout     = "15:04:05.999999999 -07:00"
	RawTimeLayout    = "2006-01-02T15:04:05Z"
)

var (
	rxDateTimeTz = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d(?:\.\d+)?)\s*([\-\+]?)(\d\d?):(\d\d)\s*$`)
	rxDateTime   = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d(?:\.\d+)?)\s*$`)
	rxDateOnly   = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d)\s*$`)
	rxTimeTz     = regexp.MustCompile(`^\s*(?:\d{4}-\d\d-\d\d )?(\d\d:\d\d:\d\d(?:\.\d+)? [-\+]\d\d:\d\d)\s*$`)
	rxTimeOnly   = regexp.MustCompile(`^\s*(?:\d{4}-\d\d-\d\d )?(\d\d:\d\d:\d\d(?:\.\d+)?)\s*$`)
	rxRawLayout  = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ)\s*$`)
)

func ParseAnyDateTime(s string) (time.Time, error) {
	if m := rxDateTimeTz.FindStringSubmatch(s); m != nil {
		return time.Parse(DateTimeTzLayout,
			fmt.Sprintf("%s %s%02s:%02s", m[1], m[2], m[3], m[4]))
	}
	if m := rxDateTime.FindStringSubmatch(s); m != nil {
		return time.Parse(DateTimeLayout, m[1])
	}
	if m := rxDateOnly.FindStringSubmatch(s); m != nil {
		return time.Parse(DateOnlyLayout, m[1])
	}
	if m := rxTimeTz.FindStringSubmatch(s); m != nil {
		return time.Parse(TimeTzLayout, m[1])
	}
	if m := rxTimeOnly.FindStringSubmatch(s); m != nil {
		return time.Parse(TimeOnlyLayout, m[1])
	}
	if m := rxRawLayout.FindStringSubmatch(s); m != nil {
		return time.Parse(RawTimeLayout, m[1])
	}
	return time.Time{}, ErrNotTimeFormat
}

var registry = map[string]*Entry{}

func (e *Entry) Register(name string) {
	registry[strings.ToUpper(name)] = e
}

func Find(name string) (*Entry, bool) {
	r, ok := registry[strings.ToUpper(name)]
	return r, ok
}

func Each(yield func(string, *Entry) bool) {
	for key, val := range registry {
		if !yield(key, val) {
			break
		}
	}
}

func findFromArgs(args []string) (*Entry, []string, error) {
	if len(args) <= 0 {
		return nil, nil, ErrTooFewArguments
	}
	spec, ok := Find(args[0])
	if ok {
		if len(args) < 2 {
			return nil, nil, ErrDSNStringIsNotSpecified
		}
		return spec, []string{args[0], strings.Join(args[1:], " ")}, nil
	}
	scheme, _, ok := strings.Cut(args[0], ":")
	if ok {
		spec, ok = Find(scheme)
		if ok {
			return spec, []string{scheme, strings.Join(args, " ")}, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: %s", ErrSupportDriveNotFound, args[0])
}

type DBInfo struct {
	Driver     string
	DataSource string
	Dialect    *Entry
}

func ReadDBInfoFromArgs(args []string) (*DBInfo, error) {
	entry, args, err := findFromArgs(args)
	if err != nil {
		return nil, err
	}
	d := &DBInfo{
		Driver:     args[0],
		DataSource: args[1],
		Dialect:    entry,
	}
	if entry.DSNFilter != nil {
		if d.DataSource, err = entry.DSNFilter(d.DataSource); err != nil {
			return nil, err
		}
	}
	return d, nil
}

type PlaceHolderQuestion struct {
	values []any
}

func (ph *PlaceHolderQuestion) Make(v any) string {
	ph.values = append(ph.values, v)
	return "?"
}

func (ph *PlaceHolderQuestion) Values() (result []any) {
	result = ph.values
	ph.values = ph.values[:0]
	return
}

type PlaceHolderName struct {
	Prefix string
	Format string
	values []any
}

func (ph *PlaceHolderName) Make(v any) string {
	ph.values = append(ph.values, v)
	return fmt.Sprintf("%s%s%d", ph.Prefix, ph.Format, len(ph.values))
}

func (ph *PlaceHolderName) Values() (result []any) {
	for i, v := range ph.values {
		result = append(result, sql.Named(fmt.Sprintf("%s%d", ph.Format, i+1), v))
	}
	ph.values = ph.values[:0]
	return
}
