package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hymkor/sqlbless"
	_ "github.com/hymkor/sqlbless/sqlite"
)

func main() {
	sqlbless.WriteSignature(os.Stdout)

	flag.Usage = sqlbless.Usage

	flag.Parse()
	if err := sqlbless.Main(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}