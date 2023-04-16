package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

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

func doSelect(ctx context.Context, conn canQuery, query string, w io.Writer) error {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("Query: %[1]w (%[1]T)", err)
	}
	defer rows.Close()
	return dumpRows(ctx, rows, ',', false, w)
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

func tee(console, spool *os.File) io.Writer {
	if spool != nil {
		return io.MultiWriter(console, spool)
	} else {
		return console
	}
}

func echo(spool *os.File, query string) {
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

func desc(ctx context.Context, conn canQuery, options *Options, table string, w io.Writer) error {
	if options.SqlForDesc == "" {
		return errors.New("DESC: not supported")
	}
	// fmt.Fprintln(os.Stderr, options.SqlForDesc)
	tableName := strings.TrimSpace(table)
	var rows *sql.Rows
	var err error
	if tableName == "" {
		rows, err = conn.QueryContext(ctx, options.SqlForTab)
	} else {
		rows, err = conn.QueryContext(ctx, options.SqlForDesc, tableName)
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	return dumpRows(ctx, rows, ',', false, w)
}

func loop(ctx context.Context, options *Options, conn *sql.DB) error {
	disabler := colorable.EnableColorsStdout(nil)
	defer disabler()

	var editor multiline.Editor
	var history History
	editor.LineEditor.History = &history
	editor.LineEditor.Writer = colorable.NewColorableStdout()

	editor.LineEditor.Coloring = &Coloring{}

	editor.Prompt = func(w io.Writer, i int) (int, error) {
		io.WriteString(w, "\x1B[0m")
		if i <= 0 {
			return io.WriteString(w, "SQL> ")
		}
		return fmt.Fprintf(w, "%3d> ", i+1)
	}
	var spool *os.File = nil
	var tx *sql.Tx = nil
	defer func() {
		if tx != nil {
			txRollback(&tx, tee(os.Stderr, spool))
		}
		if spool != nil {
			spool.Close()
			spool = nil
		}
	}()
	for {
		if spool != nil {
			fmt.Fprintf(os.Stderr, "Spooling to '%s' now\n", spool.Name())
		}
		lines, err := editor.Read(ctx)
		if err != nil {
			return err
		}
		query := trimSemicolon(strings.TrimSpace(strings.Join(lines, "\n")))
		history.Add(query)
		cmd, arg := cutField(query)
		switch strings.ToUpper(cmd) {
		case "SPOOL":
			fname, _ := cutField(arg)
			if fname == "" {
				if spool != nil {
					fmt.Fprintf(os.Stderr, "Spooling to '%s' now\n", spool.Name())
				} else {
					fmt.Fprintln(os.Stderr, "Not Spooling")
				}
				continue
			}
			if spool != nil {
				spool.Close()
				fmt.Fprintln(os.Stderr, "Spool closed.")
				spool = nil
			}
			if !strings.EqualFold(fname, "off") {
				if fd, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					spool = fd
					fmt.Fprintf(os.Stderr, "Spool to %s\n", fname)
				}
			}
		case "SELECT":
			echo(spool, query)
			if tx == nil {
				err = doSelect(ctx, conn, query, tee(os.Stdout, spool))
			} else {
				err = doSelect(ctx, tx, query, tee(os.Stdout, spool))
			}
		case "DELETE", "INSERT", "UPDATE":
			echo(spool, query)
			err = txBegin(ctx, conn, &tx, tee(os.Stderr, spool))
			if err == nil {
				err = doDML(ctx, tx, query, tee(os.Stdout, spool))
				if err != nil && !options.DontRollbackOnFail {
					fmt.Fprintln(tee(os.Stderr, spool), err.Error())
					echo(spool, "( rollback automatically )")
					err = txRollback(&tx, tee(os.Stderr, spool))
				}
			}
		case "COMMIT":
			echo(spool, query)
			err = txCommit(&tx, tee(os.Stderr, spool))
		case "ROLLBACK":
			echo(spool, query)
			err = txRollback(&tx, tee(os.Stderr, spool))
		case "EXIT", "QUIT":
			return io.EOF
		case "DESC", "\\D":
			echo(spool, query)
			err = desc(ctx, conn, options, arg, tee(os.Stdout, spool))
		case "HISTORY":
			echo(spool, query)
			csvw := csv.NewWriter(tee(os.Stdout, spool))
			for i, end := 0, history.Len(); i < end; i++ {
				text, stamp := history.textAndStamp(i)
				csvw.Write([]string{
					strconv.Itoa(i),
					stamp.Local().Format(time.DateTime),
					text})
			}
			csvw.Flush()
		default:
			echo(spool, query)
			if tx != nil {
				fmt.Fprintln(os.Stderr, "Transaction is not closed. Please Commit or Rollback.")
				continue
			}
			_, err = conn.ExecContext(ctx, query)
		}
		if err != nil {
			fmt.Fprintln(tee(os.Stderr, spool), err.Error())
		}
	}
}

var version string

func mains(args []string) error {
	fmt.Printf("SQL-Bless %s-%s-%s by %s\n",
		version, runtime.GOOS, runtime.GOARCH, runtime.Version())
	if len(args) < 2 {
		usage(os.Stdout)
		return nil
	}
	fmt.Println("  Ctrl-M or      Enter: Insert Linefeed")
	fmt.Println("  Ctrl-J or Ctrl-Enter: Exec command")
	fmt.Println()
	conn, err := sql.Open(args[0], args[1])
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer conn.Close()

	var options *Options
	options, ok := dbDependent[strings.ToUpper(args[0])]
	if !ok {
		options = &Options{}
	}
	return loop(context.Background(), options, conn)
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
