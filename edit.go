package sqlbless

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
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

	var conn canQuery
	if ss.tx == nil {
		conn = ss.conn
	} else {
		conn = ss.tx
	}

	const (
		Success = iota
		Failure
		AbortAll
	)
	getKey := pilot.GetKey
	status := Success

	editor := &spread.Editor{
		Viewer:    newViewer(ss),
		Query:     conn.QueryContext,
		DBDialect: ss.dbDialect,
		Exec: func(ctx context.Context, dmlSql string) (err error) {
			if status == Failure {
				if ans, _ := continueOrAbort(getKey); ans {
					status = Success
				} else {
					status = AbortAll
				}
			}

			if status == AbortAll {
				echoPrefix(tee(os.Stderr, ss.spool), "(cancel) ", dmlSql)
			} else {
				err = askSqlAndExecute(ctx, ss, getKey, dmlSql)
				if err != nil {
					status = Failure
					err = nil
				}
			}
			return
		},
		Auto: pilot.AutoPilotForCsvi(),
		Dump: func(ctx context.Context, rows *sql.Rows, w io.Writer) error {
			err := ss.DumpConfig.Dump(ctx, rows, w)
			rows.Close()
			return err
		},
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
	err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
	if err != nil {
		return err
	}
	echo(ss.spool, dmlSql)
	return doDML(ctx, ss.tx, dmlSql, tee(os.Stdout, ss.spool))
}
