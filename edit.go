package sqlbless

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

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
	if ss.DumpConfig.PrintType {
		hl = 3
	} else {
		hl = 1
	}
	return &spread.Viewer{
		HeaderLines: hl,
		Comma:       byte(ss.DumpConfig.Comma),
		Null:        ss.DumpConfig.Null,
		Spool:       ss.spool,
	}
}

func doEdit(ctx context.Context, ss *Session, command string, pilot CommandIn, out io.Writer) error {
	const (
		Success = iota
		Failure
		AbortAll
	)
	getKey := pilot.GetKey
	status := Success

	editor := &spread.Editor{
		Viewer:  newViewer(ss),
		Dialect: ss.Dialect,
		Exec: func(ctx context.Context, dmlSql string, args ...any) (rv sql.Result, err error) {
			if status == Failure {
				if ans, _ := continueOrAbort(getKey); ans {
					status = Success
				} else {
					status = AbortAll
				}
			}

			if status == AbortAll {
				echoPrefix(ss.stderr, "(cancel) ", dmlSql)
			} else {
				err = askSqlAndExecute(ctx, ss, getKey, dmlSql)
				if err != nil {
					status = Failure
					err = nil
				}
			}
			return
		},
		Dump: func(ctx context.Context, rows *sql.Rows, w io.Writer) error {
			return ss.DumpConfig.Dump(ctx, rows, w)
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
	_, tableAndWhere := cutField(command)
	return editor.Edit(ctx, tableAndWhere, out)
}

func askSqlAndExecute(ctx context.Context, ss *Session, getKey func() (string, error), dmlSql string) error {
	fmt.Print("\n---\n")
	fmt.Println(dmlSql)
	answer, err := ask2("Execute? [y/n] ", "yY", "nN", getKey)
	if err != nil {
		return err
	}
	if !answer {
		echoPrefix(ss.spool, "(cancel) ", dmlSql)
		return nil
	}
	err = txBegin(ctx, ss.conn, &ss.tx, ss.stderr)
	if err != nil {
		return err
	}
	echo(ss.spool, dmlSql)
	return doDML(ctx, ss.tx, dmlSql, ss.stdout)
}
