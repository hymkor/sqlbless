package sqlbless

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hymkor/sqlbless/misc"
	"github.com/hymkor/sqlbless/spread"
)

type getKeyAndSize = spread.GetKeyAndSize

const (
	_ANSI_CURSOR_OFF = "\x1B[?25l"
	_ANSI_CURSOR_ON  = "\x1B[?25h"
)

func ask2(msg, yes, no string, getKey func() (string, error)) (bool, error) {
	fmt.Print(msg, _ANSI_CURSOR_ON)
	for {
		answer, err := getKey()
		if err != nil {
			return false, err
		}
		if strings.Contains(yes, answer) {
			fmt.Println(answer, _ANSI_CURSOR_OFF)
			return true, nil
		}
		if strings.Contains(no, answer) {
			fmt.Println(answer, _ANSI_CURSOR_OFF)
			return false, nil
		}
	}
}

func continueOrAbort(getKey func() (string, error)) (bool, error) {
	return ask2("Continue or abort [c/a] ", "cC", "aA", getKey)
}

func newViewer(ss *Session) *spread.Viewer {
	var hl int
	if ss.Debug {
		hl = 3
	} else {
		hl = 1
	}
	return &spread.Viewer{
		HeaderLines: hl,
		Comma:       byte(ss.Comma),
		Null:        ss.Null,
		Spool:       ss.spool,
	}
}

func doEdit(ctx context.Context, ss *Session, command string, pilot CommandIn) error {
	const (
		Success = iota
		Failure
		AbortAll
	)
	getKey := pilot.GetKey
	status := Success

	editor := &spread.Editor{
		Viewer: newViewer(ss),
		Entry:  ss.Dialect,
		Exec: func(ctx context.Context, dmlSql string, args ...any) (rv sql.Result, err error) {
			if status == Failure {
				if ans, _ := continueOrAbort(getKey); ans {
					status = Success
				} else {
					status = AbortAll
				}
			}

			if status == AbortAll {
				echoPrefix(ss.stdErr, "(cancel) ", dmlSql)
			} else {
				err = askSqlAndExecute(ctx, ss, getKey, dmlSql, args)
				if err != nil {
					echoPrefix(ss.stdErr, "(error) ", err.Error())
					status = Failure
					err = nil
				}
			}
			return
		},
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
			fmt.Fprintf(&b, "(%s) %#v\n", n.Name, n.Value)
		} else {
			fmt.Fprintf(&b, "(%d) %#v\n", i+1, v)
		}
	}
	return b.String()
}

func askSqlAndExecute(ctx context.Context, ss *Session, getKey func() (string, error), dmlSql string, args []any) error {
	fmt.Print("\n---\n")
	fmt.Println(dmlSql)
	argsString := joinAny(args)
	if argsString != "" {
		fmt.Println(argsString)
	}
	answer, err := ask2("Execute? [y/n] ", "yY", "nN", getKey)
	if err != nil {
		return err
	}
	if !answer {
		echoPrefix(ss.spool, "(cancel) ", dmlSql)
		if argsString != "" {
			echoPrefix(ss.spool, "(args)", argsString)
		}
		return nil
	}
	isNewTx := (ss.tx == nil)
	err = txBegin(ctx, ss.conn, &ss.tx, ss.stdErr)
	if err != nil {
		return err
	}
	echo(ss.spool, dmlSql)
	if argsString != "" {
		echo(ss.spool, argsString)
	}
	count, err := doDML(ctx, ss.tx, dmlSql, args, ss.stdOut)
	if (err != nil || count == 0) && isNewTx && ss.tx != nil {
		ss.tx.Rollback()
		ss.tx = nil
	}
	return err
}
