package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/hymkor/sqlbless"
	"github.com/hymkor/sqlbless/dialect/sqlite"
)

func mains() error {
	cfg := sqlbless.New().Bind(flag.CommandLine)
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		return errors.New("Usage: sqlite3bless {DBPATH or :memory:}")
	}
	return cfg.Run("sqlite3", args[0], sqlite.Entry)
}

func main() {
	if err := mains(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
