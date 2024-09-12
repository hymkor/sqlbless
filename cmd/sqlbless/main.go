package main

import (
	"os"
	"fmt"
	"flag"

	"github.com/hymkor/sqlbless"
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
