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
	rxDateTime = regexp.MustCompile(`\d{4}-\d\d-\d\d \d\d:\d\d:\d\d`)
	rxDateOnly = regexp.MustCompile(`\d{4}-\d\d-\d\d`)
	rxTimeOnly = regexp.MustCompile(`\d\d:\d\d:\d\d`)
)

func parseAnyDateTime(s string) (time.Time, error) {
	if m := rxDateTime.FindString(s); m != "" {
		return time.Parse(dateTimeFormat, m)
	}
	if m := rxDateOnly.FindString(s); m != "" {
		return time.Parse(dateOnlyFormat, m)
	}
	if m := rxTimeOnly.FindString(s); m != "" {
		return time.Parse(timeOnlyFormat, m)
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
