package spread

import (
	"context"
	"database/sql"
	"io"
)

func tee(console, spool io.Writer) io.Writer {
	if spool != nil {
		return io.MultiWriter(console, spool)
	} else {
		return console
	}
}

type canQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
