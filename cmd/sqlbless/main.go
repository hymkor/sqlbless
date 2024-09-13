package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hymkor/sqlbless"
	_ "github.com/hymkor/sqlbless/mysql"
	_ "github.com/hymkor/sqlbless/oracle"
	_ "github.com/hymkor/sqlbless/postgresql"
	_ "github.com/hymkor/sqlbless/sqlite"
	_ "github.com/hymkor/sqlbless/sqlserver"
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
