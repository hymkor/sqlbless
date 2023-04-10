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

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/simplehistory"

	"github.com/hymkor/go-multiline-ny"
)

type Coloring struct {
	bits int
}

var (
	red   = readline.SGR3(1, 49, 31)
	cyan  = readline.SGR3(1, 49, 36)
	reset = readline.SGR3(22, 49, 39)
)

func (c *Coloring) Init() int {
	c.bits = 0
	return reset
}

func (c *Coloring) Next(r rune) int {
	const (
		_QUOTED = 1
	)
	newbits := c.bits
	if r == '\'' {
		newbits ^= _QUOTED
	}
	defer func() {
		c.bits = newbits
	}()
	if (c.bits&_QUOTED) != 0 || (newbits&_QUOTED) != 0 {
		return red
	}
	return cyan
}

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
		return fmt.Errorf("Query: %w", err)
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
		return fmt.Errorf("Exec: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("RowsAffected: %w", err)
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
		fmt.Fprintln(os.Stderr, "Rollback complete.")
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
		return fmt.Errorf("BeginTx: %w", err)
	}
	return nil
}

func loop(ctx context.Context, conn *sql.DB) error {
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
		case "SELECT":
			if tx == nil {
				err = doSelect(ctx, conn, query, os.Stdout)
			} else {
				err = doSelect(ctx, tx, query, os.Stdout)
			}
		case "DELETE", "INSERT", "UPDATE":
			err = txBegin(ctx, conn, &tx, os.Stderr)
			if err == nil {
				err = doDML(ctx, tx, query, os.Stdout)
			}
		case "COMMIT":
			err = txCommit(&tx, os.Stderr)
		case "ROLLBACK":
			err = txRollback(&tx, os.Stderr)
		case "EXIT", "QUIT":
			err = txCommit(&tx, os.Stderr)
			if err != nil {
				return err
			}
			return io.EOF
		default:
			if tx != nil {
				fmt.Fprintln(os.Stderr, "Transaction is not closed. Please Commit or Rollback.")
				continue
			}
			_, err = conn.ExecContext(ctx, query)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
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
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer conn.Close()

	return loop(context.Background(), conn)
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
