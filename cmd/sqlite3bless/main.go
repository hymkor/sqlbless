package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/hymkor/sqlbless"
	"github.com/hymkor/sqlbless/sqlite"
)

func mains() error {
	cfgSetup := sqlbless.NewConfigFromFlag()
	flag.Parse()
	args := flag.Args()
	cfg := cfgSetup()
	if len(args) < 1 {
		return errors.New("Usage: litbless {DBPATH or :memory:}")
	}
	return cfg.Run("sqlite3", args[0], sqlite.Dialect)
}

func main() {
	if err := mains(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
