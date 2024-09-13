package main

import (
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
	if err := sqlbless.Main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
