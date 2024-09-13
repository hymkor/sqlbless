package sqlbless

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type DBSpec struct {
	Usage                 string
	SqlForDesc            string
	SqlForTab             string
	DisplayDateTimeLayout string
	TypeNameToConv        func(string) func(string) (string, error)
	DSNFilter             func(string) (string, error)
}

func (dbSpec *DBSpec) TryTypeNameToConv(typeName string) func(string) (string, error) {
	if dbSpec.TypeNameToConv == nil {
		return nil
	}
	return dbSpec.TypeNameToConv(typeName)
}

const (
	DateTimeTzLayout = "2006-01-02 15:04:05.999999999 -07:00"
	DateTimeLayout   = "2006-01-02 15:04:05.999999999"
	DateOnlyLayout   = "2006-01-02"
	TimeOnlyLayout   = "15:04:05.999999999"
	TimeTzLayout     = "15:04:05.999999999 -07:00"
)

var (
	rxDateTimeTz = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d(?:\.\d+)?)\s*([\-\+]?)(\d\d?):(\d\d)\s*$`)
	rxDateTime   = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d(?:\.\d+)?)\s*$`)
	rxDateOnly   = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d)\s*$`)
	rxTimeTz     = regexp.MustCompile(`^\s*(?:\d{4}-\d\d-\d\d )?(\d\d:\d\d:\d\d(?:\.\d+)? [-\+]\d\d:\d\d)\s*$`)
	rxTimeOnly   = regexp.MustCompile(`^\s*(?:\d{4}-\d\d-\d\d )?(\d\d:\d\d:\d\d(?:\.\d+)?)\s*$`)
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
	return time.Time{}, errors.New("not time format")
}

var dbSpecs = map[string]*DBSpec{
	// "POSTGRES":  postgresSpec,
	// "ORACLE":    oracleSpec,
	// "SQLSERVER": sqlServerSpec,
	// "MYSQL":     mySqlSpec,
	// "SQLITE3":   sqliteSpec,
}

func RegisterDB(name string, setting *DBSpec) {
	dbSpecs[strings.ToUpper(name)] = setting
}
