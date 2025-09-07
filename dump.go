package sqlbless

import (
	"database/sql"
)

type unreadRows struct {
	*sql.Rows
	unread bool
}

func rowsHasNext(r *sql.Rows) (*unreadRows, bool) {
	if !r.Next() {
		return nil, false
	}
	return &unreadRows{
		Rows:   r,
		unread: true,
	}, true
}

func (r *unreadRows) Next() bool {
	if r.unread {
		r.unread = false
		return true
	}
	return r.Rows.Next()
}
