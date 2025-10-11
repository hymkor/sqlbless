package main

import (
	"fmt"
	"os"

	"github.com/hymkor/sqlbless"
	_ "github.com/hymkor/sqlbless/dialect/mysql"
	_ "github.com/hymkor/sqlbless/dialect/oracle"
	_ "github.com/hymkor/sqlbless/dialect/postgresql"
	_ "github.com/hymkor/sqlbless/dialect/sqlite"
	_ "github.com/hymkor/sqlbless/dialect/sqlserver"
)

func main() {
	if err := sqlbless.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
