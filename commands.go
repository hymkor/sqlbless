package sqlbless

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/hymkor/sqlbless/internal/misc"
)

func doSelect(ctx context.Context, ss *session, query string) error {
	var rows *sql.Rows
	var err error
	if ss.tx != nil {
		rows, err = ss.tx.QueryContext(ctx, query)
	} else {
		rows, err = ss.conn.QueryContext(ctx, query)
	}
	if err != nil {
		return fmt.Errorf("query: %[1]w (%[1]T)", err)
	}
	_rows, ok := misc.RowsHasNext(rows)
	if !ok {
		rows.Close()
		return ErrNoDataFound
	}
	v := newViewer(ss)
	if ss.automatic() {
		v.Pilot = misc.CsviNoOperation{}
	}
	return v.View(ctx, query, _rows, ss.termOut)
}

type canExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func doDML(ctx context.Context, conn canExec, query string, args []any, w io.Writer) (int64, error) {
	result, err := conn.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("exec: %[1]w (%[1]T)", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("RowsAffected: %[1]w (%[1]T)", err)
	}
	fmt.Fprintf(w, "%d record(s) updated.\n", count)
	return count, nil
}

func txCommit(tx **sql.Tx, w io.Writer) error {
	var err error
	if *tx != nil {
		err = (*tx).Commit()
		*tx = nil
	}
	if err == nil {
		fmt.Fprintln(w, "Commit complete.")
	}
	return err
}

func txRollback(tx **sql.Tx, w io.Writer) error {
	var err error
	if *tx != nil {
		err = (*tx).Rollback()
		*tx = nil
	}
	if err == nil {
		fmt.Fprintln(w, "Rollback complete.")
	}
	return err
}

func (ss *session) beginTx(ctx context.Context, w io.Writer) error {
	if ss.tx != nil {
		return nil
	}
	fmt.Fprintln(w, "Starts a transaction")
	var err error
	ss.tx, err = ss.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("BeginTx: %[1]w (%[1]T)", err)
	}
	return nil
}

func doDesc(ctx context.Context, ss *session, table string) error {
	tableName := strings.TrimSpace(table)
	var query string
	if tableName == "" {
		if ss.Dialect.SqlForTab == "" {
			return fmt.Errorf("desc: %w", ErrNotSupported)
		}
		query = ss.Dialect.SqlForTab
	} else {
		if ss.Dialect.SqlForDesc == "" {
			return fmt.Errorf("desc table: %w", ErrNotSupported)
		}
		query = strings.ReplaceAll(ss.Dialect.SqlForDesc, "{table_name}", tableName)
	}
	if ss.Debug {
		fmt.Println(query)
	}
	return doSelect(ctx, ss, query)
}
