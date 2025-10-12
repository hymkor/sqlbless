package misc

import (
	"database/sql"
)

type UnreadRows struct {
	*sql.Rows
	unread bool
}

func RowsHasNext(r *sql.Rows) (*UnreadRows, bool) {
	if !r.Next() {
		return nil, false
	}
	return &UnreadRows{
		Rows:   r,
		unread: true,
	}, true
}

func (r *UnreadRows) Next() bool {
	if r.unread {
		r.unread = false
		return true
	}
	return r.Rows.Next()
}
