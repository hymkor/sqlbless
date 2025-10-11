package sqlbless

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/hymkor/sqlbless/internal/misc"
	"github.com/hymkor/sqlbless/spread"
)

type getKeyAndSize = spread.GetKeyAndSize

const (
	_ANSI_CURSOR_OFF = "\x1B[?25l"
	_ANSI_CURSOR_ON  = "\x1B[?25h"
)

func askN(msg string, getKey func() (string, error), options ...string) (int, error) {
	fmt.Print(msg, _ANSI_CURSOR_ON)
	for {
		answer, err := getKey()
		if err != nil {
			return -1, err
		}
		for i, opt := range options {
			if strings.Contains(opt, answer) {
				fmt.Println(answer, _ANSI_CURSOR_OFF)
				return i, nil
			}
		}
	}
}

func newViewer(ss *session) *spread.Viewer {
	var hl int
	if ss.Debug {
		hl = 3
	} else {
		hl = 1
	}
	return &spread.Viewer{
		HeaderLines: hl,
		Comma:       ss.comma(),
		Null:        ss.Null,
		Spool:       ss.spool,
	}
}

func doEdit(ctx context.Context, ss *session, command string, pilot commandIn) error {
	editor := &spread.Editor{
		Viewer: &spread.Viewer{
			HeaderLines: 1,
			Comma:       ss.comma(),
			Null:        ss.Null,
		},
		Entry: ss.Dialect,
		Exec:  (&askSqlAndExecute{getKey: pilot.GetKey, session: ss}).Exec,
	}
	if a, ok := pilot.AutoPilotForCsvi(); ok {
		editor.Auto = a
	}
	if ss.tx == nil {
		editor.Query = ss.conn.QueryContext
	} else {
		editor.Query = ss.tx.QueryContext
	}

	// replace `edit ` to `select * from `
	_, tableAndWhere := misc.CutField(command)
	return editor.Edit(ctx, tableAndWhere, ss.termOut)
}

func joinAny(args []any) string {
	if len(args) <= 0 {
		return ""
	}
	var b strings.Builder
	for i, v := range args {
		if n, ok := v.(sql.NamedArg); ok {
			fmt.Fprintf(&b, "(%s) %#v ", n.Name, n.Value)
		} else {
			fmt.Fprintf(&b, "(%d) %#v ", i+1, v)
		}
	}
	return b.String()
}

type statusValue int

const (
	success statusValue = iota
	failure
	discardAll
	applyAll
)

type askSqlAndExecute struct {
	status statusValue
	getKey func() (string, error)
	*session
}

func (this *askSqlAndExecute) Exec(ctx context.Context, dmlSql string, args ...any) (sql.Result, error) {
	fmt.Print("\n---\n")
	fmt.Println(dmlSql)
	fmt.Println()
	argsString := joinAny(args)
	if argsString != "" {
		fmt.Println(argsString)
	}

	if this.status == failure {
		if n, _ := askN("Continue or abort [c/a] ", this.getKey, "cC", "aA"); n == 0 {
			this.status = success
		} else {
			this.status = discardAll
		}
	}
	if this.status == discardAll {
		misc.EchoPrefix(this.stdErr, "(cancel) ", dmlSql)
		return nil, nil
	}
	fmt.Println()
	if this.status == success {
		answer, err := askN(`Apply this change? ("y":yes, "n":no, "a":all, "N":none) `, this.getKey, "y", "n", "aA", "N")
		if err != nil {
			misc.EchoPrefix(this.stdErr, "(error) ", err.Error())
			this.status = failure
			return nil, err
		}
		switch answer {
		case 1:
			// cancel and quit with no error
			this.status = success
			misc.EchoPrefix(this.spool, "(cancel) ", dmlSql)
			if argsString != "" {
				misc.EchoPrefix(this.spool, "(args)", argsString)
			}
			return nil, nil
		case 2:
			// apply all
			this.status = applyAll
		case 3:
			// discard all and quit with no error
			this.status = discardAll
			misc.EchoPrefix(this.spool, "(cancel) ", dmlSql)
			if argsString != "" {
				misc.EchoPrefix(this.spool, "(args)", argsString)
			}
			return nil, nil
		}
	}
	isNewTx := (this.tx == nil)
	err := txBegin(ctx, this.conn, &this.tx, this.stdErr)
	if err != nil {
		return nil, err
	}
	misc.Echo(this.spool, dmlSql)
	if argsString != "" {
		misc.Echo(this.spool, argsString)
	}
	result, err := this.tx.ExecContext(ctx, dmlSql, args...)
	var count int64
	if err == nil {
		count, err = result.RowsAffected()
		if err == nil && count == 0 {
			err = errors.New("no rows affected")
		}
	}
	if err != nil && isNewTx && this.tx != nil {
		this.tx.Rollback()
		this.tx = nil
	}
	fmt.Fprintf(this.stdOut, "%d record(s) updated.\n", count)
	return result, err
}
