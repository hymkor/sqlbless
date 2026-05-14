//go:build debug

package dbg

import (
	"github.com/nyaosorg/go-windows-dbg"
)

func Println(v ...any) {
	dbg.Println(v...)
}
