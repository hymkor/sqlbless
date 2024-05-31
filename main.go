package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/mattn/go-colorable"

	"github.com/hymkor/go-multiline-ny"
	"github.com/nyaosorg/go-readline-ny/auto"
	"github.com/nyaosorg/go-readline-ny/completion"
	"github.com/nyaosorg/go-readline-ny/keys"
)

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

func doSelect(ctx context.Context, conn canQuery, query string, r2c *RowToCsv, out, spool io.Writer) error {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query: %[1]w (%[1]T)", err)
	}
	_rows := rowsHasNext(rows)
	if _rows == nil {
		rows.Close()
		return fmt.Errorf("data not found")
	}
	return csvPager(query, r2c.Comma, *flagAuto != "", func(pOut io.Writer) error {
		_err := r2c.Dump(ctx, _rows, pOut)
		rows.Close()
		return _err
	}, out, spool)
}

type canExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func doDML(ctx context.Context, conn canExec, query string, w io.Writer) error {
	result, err := conn.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("exec: %[1]w (%[1]T)", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("RowsAffected: %[1]w (%[1]T)", err)
	}
	fmt.Fprintf(w, "%d record(s) updated.\n", count)
	return nil
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

func txBegin(ctx context.Context, conn *sql.DB, tx **sql.Tx, w io.Writer) error {
	if *tx != nil {
		return nil
	}
	fmt.Fprintln(w, "Starts a transaction")
	var err error
	*tx, err = conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("BeginTx: %[1]w (%[1]T)", err)
	}
	return nil
}

func tee(console, spool io.Writer) io.Writer {
	if spool != nil {
		return io.MultiWriter(console, spool)
	} else {
		return console
	}
}

func echo(spool io.Writer, query string) {
	echoPrefix(spool, "", query)
}

func echoPrefix(spool io.Writer, prefix, query string) {
	if spool != nil {
		next := true
		fmt.Fprintf(spool, "### <%s> ###\n", time.Now().Local().Format(time.DateTime))
		query = query + ";"
		for next {
			var line string
			line, query, next = strings.Cut(query, "\n")
			fmt.Fprintf(spool, "# %s%s\n", prefix, line)
		}
	}
}

func (ss *Session) desc(ctx context.Context, table string, out, spool io.Writer) error {
	// fmt.Fprintln(os.Stderr, dbSpec.SqlForDesc)
	tableName := strings.TrimSpace(table)
	var rows *sql.Rows
	var err error
	if tableName == "" {
		if ss.dbSpec.SqlForTab == "" {
			return errors.New("DESC: not supported")
		}
		rows, err = ss.conn.QueryContext(ctx, ss.dbSpec.SqlForTab)
	} else {
		if ss.dbSpec.SqlForDesc == "" {
			return errors.New("DESC TABLE: not supported")
		}
		sql := strings.ReplaceAll(ss.dbSpec.SqlForDesc, "{table_name}", tableName)
		rows, err = ss.conn.QueryContext(ctx, sql, tableName)
	}
	if err != nil {
		return err
	}
	_rows := rowsHasNext(rows)
	if _rows == nil {
		rows.Close()
		if table == "" {
			return errors.New("no tables are found")
		}
		return fmt.Errorf("%s: table not found", table)
	}
	return csvPager(table, ss.DumpConfig.Comma, *flagAuto != "", func(pOut io.Writer) error {
		err := ss.DumpConfig.Dump(ctx, _rows, pOut)
		rows.Close()
		return err
	}, out, spool)
}

var (
	flagCrLf           = flag.Bool("crlf", false, "use CRLF")
	flagFieldSeperator = flag.String("fs", ",", "Set field separator")
	flagNullString     = flag.String("null", "<NULL>", "Set a string representing NULL")
	flagTsv            = flag.Bool("tsv", false, "Use TAB as seperator")
	flagSubmitByEnter  = flag.Bool("submit-enter", false, "Submit by [Enter] and insert a new line by [Ctrl]-[Enter]")
	flagScript         = flag.String("f", "", "script file")
	flagPrintType      = flag.Bool("print-type", false, "(for debug) Print type in CSV")
	flagAuto           = flag.String("auto", "", "autopilot")
)

type CommandIn interface {
	Read(context.Context) ([]string, error)
	GetKey() (string, error)

	// AutoPilotForCsvi returns Tty object when AutoPilot is enabled.
	// When disabled, it MUST return nil.
	AutoPilotForCsvi() getKeyAndSize
}

type Script struct {
	br   *bufio.Reader
	echo io.Writer
}

func (script *Script) GetKey() (string, error) {
	return "", io.EOF
}

func (script *Script) AutoPilotForCsvi() getKeyAndSize {
	return nil
}

func (script *Script) Read(context.Context) ([]string, error) {
	var buffer strings.Builder
	quoted := 0
	for {
		ch, _, err := script.br.ReadRune()
		if err != nil {
			code := buffer.String()
			fmt.Fprintln(script.echo, code)
			return []string{code}, err
		}
		switch ch {
		case '\r':
		case '\'':
			quoted ^= 1
			buffer.WriteByte('\'')
		case '"':
			quoted ^= 2
			buffer.WriteByte('"')
		case ';':
			if quoted == 0 {
				code := buffer.String()
				fmt.Fprintln(script.echo, code)
				return []string{code}, nil
			}
			buffer.WriteByte(';')
		default:
			buffer.WriteRune(ch)
		}
	}
}

type InteractiveIn struct {
	*multiline.Editor
	tty getKeyAndSize
}

func (i *InteractiveIn) GetKey() (string, error) {
	return i.Editor.LineEditor.Tty.GetKey()
}

func (i *InteractiveIn) AutoPilotForCsvi() getKeyAndSize {
	return i.tty
}

type Session struct {
	DumpConfig RowToCsv
	dbSpec     *DBSpec
	conn       *sql.DB
	history    *History
	tx         *sql.Tx
	spool      FilterSource
}

func (ss *Session) Close() {
	if ss.tx != nil {
		txRollback(&ss.tx, tee(os.Stderr, ss.spool))
	}
	if ss.spool != nil {
		ss.spool.Close()
		ss.spool = nil
	}
}

func (ss *Session) StartFromStdin(ctx context.Context) error {
	script := &Script{
		br:   bufio.NewReader(os.Stdin),
		echo: os.Stderr,
	}
	return ss.Loop(ctx, script, true)
}

func (ss *Session) Start(ctx context.Context, fname string) error {
	if fname == "-" {
		return ss.StartFromStdin(ctx)
	}
	fd, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer fd.Close()
	script := &Script{
		br:   bufio.NewReader(fd),
		echo: os.Stderr,
	}
	return ss.Loop(ctx, script, true)
}

func (ss *Session) Loop(ctx context.Context, commandIn CommandIn, onErrorAbort bool) error {
	for {
		if ss.spool != nil {
			fmt.Fprintf(os.Stderr, "\nSpooling to '%s' now\n", ss.spool.Name())
		}
		lines, err := commandIn.Read(ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		query := strings.TrimRight(strings.Join(lines, "\n"), "; \n\r\t\v")
		if query == "" {
			continue
		}
		if _, ok := commandIn.(*Script); !ok {
			ss.history.Add(query)
		}
		cmd, arg := cutField(query)
		switch strings.ToUpper(cmd) {
		case "REM":
			// nothing to do
		case "SPOOL":
			fname, _ := cutField(arg)
			if fname == "" {
				if ss.spool != nil {
					fmt.Fprintf(os.Stderr, "Spooling to '%s' now\n", ss.spool.Name())
				} else {
					fmt.Fprintln(os.Stderr, "Not Spooling")
				}
				continue
			}
			if ss.spool != nil {
				ss.spool.Close()
				fmt.Fprintln(os.Stderr, "Spool closed.")
				ss.spool = nil
			}
			if !strings.EqualFold(fname, "off") {
				if fd, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					if *flagCrLf {
						ss.spool = newFilter(fd)
					} else {
						ss.spool = fd
					}
					fmt.Fprintf(os.Stderr, "Spool to %s\n", fname)
					writeSignature(ss.spool)
				}
			}
		case "EDIT":
			echo(ss.spool, query)
			err = doEdit(ctx, ss, query, commandIn, os.Stdout, ss.spool)

		case "SELECT":
			echo(ss.spool, query)
			if ss.tx == nil {
				err = doSelect(ctx, ss.conn, query, &ss.DumpConfig, os.Stdout, ss.spool)
			} else {
				err = doSelect(ctx, ss.tx, query, &ss.DumpConfig, os.Stdout, ss.spool)
			}
		case "DELETE", "INSERT", "UPDATE":
			echo(ss.spool, query)
			err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
			if err == nil {
				err = doDML(ctx, ss.tx, query, tee(os.Stdout, ss.spool))
			}
		case "COMMIT":
			echo(ss.spool, query)
			err = txCommit(&ss.tx, tee(os.Stderr, ss.spool))
		case "ROLLBACK":
			echo(ss.spool, query)
			err = txRollback(&ss.tx, tee(os.Stderr, ss.spool))
		case "EXIT", "QUIT":
			return nil
		case "DESC", "\\D":
			echo(ss.spool, query)
			err = ss.desc(ctx, arg, os.Stdout, ss.spool)
		case "HISTORY":
			echo(ss.spool, query)
			csvw := csv.NewWriter(tee(os.Stdout, ss.spool))
			for i, end := 0, ss.history.Len(); i < end; i++ {
				text, stamp := ss.history.textAndStamp(i)
				csvw.Write([]string{
					strconv.Itoa(i),
					stamp.Local().Format(time.DateTime),
					text})
			}
			csvw.Flush()
		case "START":
			fname, _ := cutField(arg)
			err = ss.Start(ctx, fname)
		default:
			echo(ss.spool, query)
			if ss.tx != nil {
				err = errors.New("transaction is not closed. Please Commit or Rollback")
			} else {
				_, err = ss.conn.ExecContext(ctx, query)
				if err == nil {
					fmt.Fprintln(tee(os.Stderr, ss.spool), "Ok")
				}
			}
		}
		if err != nil {
			fmt.Fprintln(tee(os.Stderr, ss.spool), err.Error())
			if onErrorAbort {
				return err
			}
		}
	}
}

type sqlCompletion struct{}

func (sqlCompletion) Enclosures() string {
	return `'"`
}

func (sqlCompletion) Delimiters() string {
	return ","
}

func (sqlCompletion) List(field []string) (fullnames []string, basenames []string) {
	if len(field) <= 1 {
		fullnames = []string{
			"ALTER", "COMMIT", "CREATE", "DELETE", "DESC", "DROP", "EXIT",
			"HISTORY", "INSERT", "QUIT", "REM", "ROLLBACK", "SELECT", "SPOOL",
			"START", "TRUNCATE", "UPDATE", "\\D",
		}
	} else if len(field) >= 5 {
		fullnames = []string{
			"AND", "FROM", "INTO", "OR", "WHERE",
		}
	} else if len(field) >= 3 {
		fullnames = []string{
			"FROM", "INTO",
		}
	}
	return
}

func mains(args []string) error {
	if len(args) < 2 {
		usage(os.Stdout)
		return nil
	}

	dbSpec, ok := dbSpecs[strings.ToUpper(args[0])]
	if !ok {
		return fmt.Errorf("not support driver: %s", args[0])
	}

	conn, err := sql.Open(args[0], args[1])
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer conn.Close()

	if err = conn.Ping(); err != nil {
		return err
	}

	var history History

	session := &Session{
		dbSpec:  dbSpec,
		conn:    conn,
		history: &history,
	}
	defer session.Close()

	session.DumpConfig.Null = *flagNullString
	if *flagTsv {
		session.DumpConfig.Comma = '\t'
	} else {
		session.DumpConfig.Comma, _ = utf8.DecodeRuneInString(*flagFieldSeperator)
	}
	session.DumpConfig.PrintType = *flagPrintType

	ctx := context.Background()
	if *flagScript != "" {
		return session.Start(ctx, *flagScript)
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return session.StartFromStdin(ctx)
	}

	// interactive mode

	fmt.Println("  Ctrl-M or      Enter: Insert Linefeed")
	fmt.Println("  Ctrl-J or Ctrl-Enter: Exec command")
	fmt.Println()

	disabler := colorable.EnableColorsStdout(nil)
	defer disabler()

	var editor multiline.Editor

	if *flagSubmitByEnter {
		editor.SwapEnter()
	}
	var tty getKeyAndSize
	if *flagAuto != "" {
		text := strings.ReplaceAll(*flagAuto, "||", "\n") // "||" -> Ctrl-J(Commit)
		text = strings.ReplaceAll(text, "|", "\r")        // "|" -> Ctrl-M (NewLine)
		if text[len(text)-1] != '\n' {                    // EOF -> Ctrl-J(Commit)
			text = text + "\n"
		}
		tty1 := &auto.Pilot{
			Text: strings.Split(text, ""),
		}
		editor.LineEditor.Tty = tty1
		tty = tty1
	}
	editor.SetHistory(&history)
	editor.SetWriter(colorable.NewColorableStdout())
	editor.SetColoring(&Coloring{})
	editor.SetPrompt(func(w io.Writer, i int) (int, error) {
		io.WriteString(w, "\x1B[0m")
		if i <= 0 {
			return io.WriteString(w, "SQL> ")
		}
		return fmt.Fprintf(w, "%3d> ", i+1)
	})
	editor.BindKey(keys.CtrlI, completion.CmdCompletion{
		Completion: sqlCompletion{},
	})
	editor.SubmitOnEnterWhen(func(lines []string, csrline int) bool {
		for {
			last := strings.TrimSpace(lines[len(lines)-1])
			if last != "" || len(lines) <= 1 {
				c, _ := utf8.DecodeLastRuneInString(last)
				return c == ';'
			}
			lines = lines[:len(lines)-1]
		}
	})

	return session.Loop(ctx, &InteractiveIn{
		Editor: &editor,
		tty:    tty,
	}, false)
}

var version string

func writeSignature(w io.Writer) {
	fmt.Fprintf(w, "# SQL-Bless %s-%s-%s by %s\n",
		version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func main() {
	writeSignature(os.Stdout)

	flag.Parse()
	if err := mains(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
