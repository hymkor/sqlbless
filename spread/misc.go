package spread

import (
	"context"
	"database/sql"
	"io"
	"strings"
)

func tee(console, spool io.Writer) io.Writer {
	if spool != nil {
		return io.MultiWriter(console, spool)
	} else {
		return console
	}
}

func cutField(s string) (string, string) {
	s = strings.TrimLeft(s, " \n\r\t\v")
	i := 0
	for len(s) > i && s[i] != ' ' && s[i] != '\n' && s[i] != '\r' && s[i] != '\t' && s[i] != '\v' {
		i++
	}
	return s[:i], s[i:]
}

type canQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
