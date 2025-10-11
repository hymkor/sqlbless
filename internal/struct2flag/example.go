//go:build run

package main

import (
	"flag"
	"fmt"

	"github.com/hymkor/sqlbless/internal/struct2flag"
)

type Env struct {
	B bool   `flag:"b,boolean flag"`
	N int    `flag:"n,integer flag"`
	S string `flag:"s,string flag"`
}

func (e Env) Run() {
	fmt.Printf("B=%#v\n", e.B)
	fmt.Printf("N=%#v\n", e.N)
	fmt.Printf("S=%#v\n", e.S)
}

func main() {
	var env Env
	struct2flag.BindDefault(&env)
	flag.Parse()
	env.Run()
}
