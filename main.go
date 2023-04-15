package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/mattn/go-colorable"
	_ "github.com/sijms/go-ora/v2"

	"github.com/nyaosorg/go-readline-ny/simplehistory"

	"github.com/hymkor/go-multiline-ny"
)

func firstWord(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t' || s[0] == '\v') {
		s = s[1:]
	}
	i := 0
	for len(s) > i && s[i] != ' ' && s[i] != '\n' && s[i] != '\r' && s[i] != '\t' && s[i] != '\v' {
		i++
	}
	return s[:i]
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
		for next {
			var line string
			line, query, next = strings.Cut(query, "\n")
			fmt.Fprintf(spool, "# %s\n", line)
		}
	}
}

type Options struct {
	RollbackOnFail bool
}

func loop(ctx context.Context, options *Options, conn *sql.DB) error {
	disabler := colorable.EnableColorsStdout(nil)
	defer disabler()

	var editor multiline.Editor
	history := simplehistory.New()
	editor.LineEditor.History = history
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
	defer func() {
		if spool != nil {
			spool.Close()
			spool = nil
		}
	}()
	var tx *sql.Tx = nil
	for {
		lines, err := editor.Read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				txCommit(&tx, os.Stderr)
			} else {
				txRollback(&tx, os.Stderr)
			}
			return err
		}
		query := trimSemicolon(strings.TrimSpace(strings.Join(lines, "\n")))
		history.Add(query)
		switch strings.ToUpper(firstWord(query)) {
		case "SPOOL":
			if spool != nil {
				spool.Close()
				fmt.Fprintln(os.Stderr, "Spool closed.")
				spool = nil
			}
			fname := firstWord(query[5:])
			if fname != "" && !strings.EqualFold(fname, "off") {
				if fd, err := os.Create(fname); err == nil {
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
				if err != nil && options.RollbackOnFail {
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
			err = txCommit(&tx, tee(os.Stderr, spool))
			if err != nil {
				return err
			}
			return io.EOF
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

func mains(args []string) error {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, `  sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE`)
		fmt.Fprintln(os.Stderr, `  sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"`)
		return nil
	}
	fmt.Println("SQL*Bless")
	fmt.Println("  Ctrl-M or      Enter: Insert Linefeed")
	fmt.Println("  Ctrl-J or Ctrl-Enter: Exec command")
	fmt.Println()
	conn, err := sql.Open(args[0], args[1])
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer conn.Close()

	var options Options
	switch strings.ToUpper(args[0]) {
	case "POSTGRES":
		options.RollbackOnFail = true
	}
	return loop(context.Background(), &options, conn)
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
