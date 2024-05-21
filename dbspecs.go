package main

import (
	"fmt"
	"io"
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

const dateTimeFormat = "2006-01-02 15:04:05"

var dbSpecs = map[string]*DBSpec{
	"POSTGRES":  postgreSqlSpec,
	"ORACLE":    oracleSpec,
	"SQLSERVER": sqlServerSpec,
	"MYSQL":     mySqlSpec,
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	for _, d := range dbSpecs {
		fmt.Fprintf(w, "  %s\n", d.Usage)
	}
}
