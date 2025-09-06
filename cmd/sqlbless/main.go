package main

import (
	"fmt"
	"os"

	"github.com/hymkor/sqlbless"
	_ "github.com/hymkor/sqlbless/dbdialect/mysql"
	_ "github.com/hymkor/sqlbless/dbdialect/oracle"
	_ "github.com/hymkor/sqlbless/dbdialect/postgresql"
	_ "github.com/hymkor/sqlbless/dbdialect/sqlite"
	_ "github.com/hymkor/sqlbless/dbdialect/sqlserver"
)

func main() {
	if err := sqlbless.Main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
