package main

import (
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

	"github.com/mattn/go-colorable"

	"github.com/hymkor/go-multiline-ny"
)

func cutField(s string) (string, string) {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t' || s[0] == '\v') {
		s = s[1:]
	}
	i := 0
	for len(s) > i && s[i] != ' ' && s[i] != '\n' && s[i] != '\r' && s[i] != '\t' && s[i] != '\v' {
		i++
	}
	return s[:i], s[i:]
}

func trimSemicolon(s string) string {
	if len(s) >= 1 && s[len(s)-1] == ';' {
		s = s[:len(s)-1]
		s = strings.TrimSpace(s)
	}
	return s
}

type canQuery interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func doSelect(ctx context.Context, conn canQuery, query string, dcfg *DumpConfig, w io.Writer) error {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("Query: %[1]w (%[1]T)", err)
	}
	defer rows.Close()
	return dcfg.DumpRows(ctx, rows, w)
}

type canExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func doDML(ctx context.Context, conn canExec, query string, w io.Writer) error {
	result, err := conn.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("Exec: %[1]w (%[1]T)", err)
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
	if spool != nil {
		next := true
		fmt.Fprintf(spool, "# (%s)\n", time.Now().Local().Format(time.DateTime))
		for next {
			var line string
			line, query, next = strings.Cut(query, "\n")
			fmt.Fprintf(spool, "# %s\n", line)
		}
	}
}

func desc(ctx context.Context, conn canQuery, dbSpec *DBSpec, table string, dcfg *DumpConfig, w io.Writer) error {
	// fmt.Fprintln(os.Stderr, dbSpec.SqlForDesc)
	tableName := strings.TrimSpace(table)
	var rows *sql.Rows
	var err error
	if tableName == "" {
		if dbSpec.SqlForTab == "" {
			return errors.New("DESC TABLE: not supported")
		}
		rows, err = conn.QueryContext(ctx, dbSpec.SqlForTab)
	} else {
		if dbSpec.SqlForDesc == "" {
			return errors.New("DESC TABLE: not supported")
		}
		rows, err = conn.QueryContext(ctx, dbSpec.SqlForDesc, tableName)
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	return dcfg.DumpRows(ctx, rows, w)
}

var (
	flagCrLf           = flag.Bool("crlf", false, "use CRLF")
	flagFieldSeperator = flag.String("fs", ",", "Set field separator")
	flagNullString     = flag.String("null", "<NULL>", "Set a string representing NULL")
	flagTsv            = flag.Bool("tsv", false, "Use TAB as seperator")
)

type CommandIn interface {
	Read(context.Context) ([]string, error)
}

type Session struct {
	DumpConfig
	dbSpec       *DBSpec
	conn         *sql.DB
	history      *History
	onErrorAbort bool
	tx           *sql.Tx
	spool        FilterSource
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

func (ss *Session) Loop(ctx context.Context, commandIn CommandIn) error {
	for {
		if ss.spool != nil {
			fmt.Fprintf(os.Stderr, "Spooling to '%s' now\n", ss.spool.Name())
		}
		lines, err := commandIn.Read(ctx)
		if err != nil {
			return err
		}
		query := trimSemicolon(strings.TrimSpace(strings.Join(lines, "\n")))
		ss.history.Add(query)
		cmd, arg := cutField(query)
		switch strings.ToUpper(cmd) {
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
		case "SELECT":
			echo(ss.spool, query)
			if ss.tx == nil {
				err = doSelect(ctx, ss.conn, query, &ss.DumpConfig, tee(os.Stdout, ss.spool))
			} else {
				err = doSelect(ctx, ss.tx, query, &ss.DumpConfig, tee(os.Stdout, ss.spool))
			}
		case "DELETE", "INSERT", "UPDATE":
			echo(ss.spool, query)
			err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
			if err == nil {
				err = doDML(ctx, ss.tx, query, tee(os.Stdout, ss.spool))
				if err != nil {
					if ss.onErrorAbort || !ss.dbSpec.DontRollbackOnFail {
						fmt.Fprintln(tee(os.Stderr, ss.spool), err.Error())
						echo(ss.spool, "( rollback automatically )")
						errRollback := txRollback(&ss.tx, tee(os.Stderr, ss.spool))
						if ss.onErrorAbort {
							return err
						}
						err = errRollback
					}
				}
			}
		case "COMMIT":
			echo(ss.spool, query)
			err = txCommit(&ss.tx, tee(os.Stderr, ss.spool))
		case "ROLLBACK":
			echo(ss.spool, query)
			err = txRollback(&ss.tx, tee(os.Stderr, ss.spool))
		case "EXIT", "QUIT":
			return io.EOF
		case "DESC", "\\D":
			echo(ss.spool, query)
			err = desc(ctx, ss.conn, ss.dbSpec, arg, &ss.DumpConfig, tee(os.Stdout, ss.spool))
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
		default:
			echo(ss.spool, query)
			if ss.tx != nil {
				fmt.Fprintln(os.Stderr, "Transaction is not closed. Please Commit or Rollback.")
				continue
			}
			_, err = ss.conn.ExecContext(ctx, query)
		}
		if err != nil {
			fmt.Fprintln(tee(os.Stderr, ss.spool), err.Error())
			if ss.onErrorAbort {
				return err
			}
		}
	}
}

func mains(args []string) error {
	if len(args) < 2 {
		usage(os.Stdout)
		return nil
	}
	conn, err := sql.Open(args[0], args[1])
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer conn.Close()

	if err = conn.Ping(); err != nil {
		return err
	}

	dbSpec, ok := dbSpecs[strings.ToUpper(args[0])]
	if !ok {
		dbSpec = &DBSpec{}
	}

	var history History

	session := &Session{
		dbSpec:       dbSpec,
		conn:         conn,
		history:      &history,
		onErrorAbort: false,
	}
	defer session.Close()

	session.DumpConfig.Null = *flagNullString
	if *flagTsv {
		session.DumpConfig.Comma = '\t'
	} else {
		session.DumpConfig.Comma, _ = utf8.DecodeRuneInString(*flagFieldSeperator)
	}

	// interactive mode

	fmt.Println("  Ctrl-M or      Enter: Insert Linefeed")
	fmt.Println("  Ctrl-J or Ctrl-Enter: Exec command")
	fmt.Println()

	disabler := colorable.EnableColorsStdout(nil)
	defer disabler()

	var editor multiline.Editor

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

	return session.Loop(context.Background(), &editor)
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
