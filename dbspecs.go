package main

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"
)

type DBSpec struct {
	Usage          string
	SqlForDesc     string
	SqlForTab      string
	TypeNameToConv func(string) func(string) (string, error)
}

func (dbSpec *DBSpec) TryTypeNameToConv(typeName string) func(string) (string, error) {
	if dbSpec.TypeNameToConv == nil {
		return nil
	}
	return dbSpec.TypeNameToConv(typeName)
}

const (
	dateTimeFormat = "2006-01-02 15:04:05"
	dateOnlyFormat = "2006-01-02"
	timeOnlyFormat = "15:04:05"
)

var (
	rxDateTime = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d)\s*$`)
	rxDateOnly = regexp.MustCompile(`^\s*(\d{4}-\d\d-\d\d)\s*$`)
	rxTimeOnly = regexp.MustCompile(`^\s*(\d\d:\d\d:\d\d)\s*$`)
)

func parseAnyDateTime(s string) (time.Time, error) {
	if m := rxDateTime.FindStringSubmatch(s); m != nil {
		return time.Parse(dateTimeFormat, m[1])
	}
	if m := rxDateOnly.FindStringSubmatch(s); m != nil {
		return time.Parse(dateOnlyFormat, m[1])
	}
	if m := rxTimeOnly.FindStringSubmatch(s); m != nil {
		return time.Parse(timeOnlyFormat, m[1])
	}
	return time.Time{}, errors.New("not time format")
}

var dbSpecs = map[string]*DBSpec{
	"POSTGRES":  postgreSqlSpec,
	"ORACLE":    oracleSpec,
	"SQLSERVER": sqlServerSpec,
	"MYSQL":     mySqlSpec,
	"SQLITE3":   sqliteSpec,
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	for _, d := range dbSpecs {
		fmt.Fprintf(w, "  %s\n", d.Usage)
	}
}
