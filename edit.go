package sqlbless

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/nyaosorg/go-box/v3"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/spread"

	"github.com/hymkor/sqlbless/internal/misc"
)

const (
	_ANSI_CURSOR_OFF = "\x1B[?25l"
	_ANSI_CURSOR_ON  = "\x1B[?25h"
)

type ErrColumnNotFound string

func (e ErrColumnNotFound) Error() string {
	return fmt.Sprintf("%s: column not found", string(e))
}

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

var rxNonQuote = regexp.MustCompile(`^\w+$`)

func chooseTable(ctx context.Context, tables []string, d *dialect.Entry, ttyout io.Writer) (string, error) {
	fmt.Fprintln(ttyout, "Select a table:")
	table, err := box.SelectString(tables, false, ttyout)
	fmt.Println()
	if err != nil {
		return "", err
	}
	if len(table) < 1 {
		return "", nil
	}
	targetTable := table[0]
	if !rxNonQuote.MatchString(targetTable) {
		targetTable = d.EncloseIdentifier(table[0])
	}
	return targetTable, nil
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
		editor.Pilot = misc.AutoCsvi{GetKeyAndSize: a}
	}
	if ss.tx == nil {
		editor.Query = ss.conn.QueryContext
	} else {
		editor.Query = ss.tx.QueryContext
	}
	// replace `edit ` to `select * from `
	_, tableAndWhere := misc.CutField(command)
	if tableAndWhere == "" {
		tables, err := ss.Dialect.FetchTables(ctx, ss.conn)
		if err != nil {
			return err
		}
		tableAndWhere, err = chooseTable(ctx, tables, ss.Dialect, ss.termOut)
		if err != nil || tableAndWhere == "" {
			return err
		}
	}
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

func (ss *askSqlAndExecute) Exec(ctx context.Context, dmlSql string, args ...any) (sql.Result, error) {
	fmt.Print("\n---\n")
	fmt.Println(dmlSql)
	fmt.Println()
	argsString := joinAny(args)
	if argsString != "" {
		fmt.Println(argsString)
	}

	if ss.status == failure {
		if n, _ := askN("Continue or abort [c/a] ", ss.getKey, "cC", "aA"); n == 0 {
			ss.status = success
		} else {
			ss.status = discardAll
		}
	}
	if ss.status == discardAll {
		misc.EchoPrefix(ss.stdErr, "(cancel) ", dmlSql)
		return nil, nil
	}
	fmt.Println()
	if ss.status == success {
		answer, err := askN(`Apply this change? ("y":yes, "n":no, "a":all, "N":none) `, ss.getKey, "y", "n", "aA", "N")
		if err != nil {
			misc.EchoPrefix(ss.stdErr, "(error) ", err.Error())
			ss.status = failure
			return nil, err
		}
		switch answer {
		case 1:
			// cancel and quit with no error
			ss.status = success
			misc.EchoPrefix(ss.spool, "(cancel) ", dmlSql)
			if argsString != "" {
				misc.EchoPrefix(ss.spool, "(args)", argsString)
			}
			return nil, nil
		case 2:
			// apply all
			ss.status = applyAll
		case 3:
			// discard all and quit with no error
			ss.status = discardAll
			misc.EchoPrefix(ss.spool, "(cancel) ", dmlSql)
			if argsString != "" {
				misc.EchoPrefix(ss.spool, "(args)", argsString)
			}
			return nil, nil
		}
	}
	isNewTx := (ss.tx == nil)
	err := ss.beginTx(ctx, ss.stdErr)
	if err != nil {
		return nil, err
	}
	misc.Echo(ss.spool, dmlSql)
	if argsString != "" {
		misc.Echo(ss.spool, argsString)
	}
	result, err := ss.tx.ExecContext(ctx, dmlSql, args...)
	var count int64
	if err == nil {
		count, err = result.RowsAffected()
		if err == nil && count == 0 {
			err = ErrNoDataFound
		}
	}
	if err != nil && isNewTx && ss.tx != nil {
		ss.tx.Rollback()
		ss.tx = nil
	}
	fmt.Fprintf(ss.stdOut, "%d record(s) updated.\n", count)
	return result, err
}
