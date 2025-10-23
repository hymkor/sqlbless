package dialect

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"database/sql"
)

type PlaceHolder interface {
	Make(any) string
	Values() []any
}

type Entry struct {
	Usage                 string
	SqlForDesc            string
	SqlForTab             string
	TableField            string
	ColumnField           string
	DisplayDateTimeLayout string
	PlaceHolder           PlaceHolder
	TypeNameToConv        func(string) func(string) (any, error)
	DSNFilter             func(string) (string, error)
	CanUseInTransaction   func(string) bool
	IsQuerySQL            func(string) bool
}

func (D *Entry) TypeToConv(typeName string) func(string) (any, error) {
	if D.TypeNameToConv == nil {
		return nil
	}
	return D.TypeNameToConv(typeName)
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
	return time.Time{}, errors.New("not time format")
}

var registry = map[string]*Entry{}

func Register(name string, setting *Entry) {
	registry[strings.ToUpper(name)] = setting
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
		return nil, nil, errors.New("too few arguments")
	}
	spec, ok := Find(args[0])
	if ok {
		if len(args) < 2 {
			return nil, nil, errors.New("DSN String is not specified")
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
	return nil, nil, fmt.Errorf("support driver not found: %s", args[0])
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
