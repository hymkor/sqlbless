package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	_ "github.com/sijms/go-ora/v2"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/simplehistory"

	"github.com/hymkor/go-multiline-ny"
)

type Container struct {
	Value any
}

func (s *Container) Scan(val any) error {
	s.Value = val
	return nil
}

func dumpRows(ctx context.Context, rows *sql.Rows, fs, rs string, w io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("(sql.Rows) Columns: %w", err)
	}
	fmt.Fprintln(w, strings.Join(columns, ","))
	item := make([]interface{}, len(columns))
	for i := 0; i < len(item); i++ {
		item[i] = &Container{}
	}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := rows.Scan(item...); err != nil {
			return fmt.Errorf("(sql.Rows) Scan: %w", err)
		}
		for i, item1 := range item {
			if i > 0 {
				io.WriteString(w, fs)
			}
			if sc, ok := item1.(*Container); ok {
				fmt.Fprintf(w, "%#v", sc.Value)
			}
		}
		io.WriteString(w, rs)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("(sql.Rows) Err: %w", err)
	}
	return nil
}

type Coloring struct {
	bits int
}

func (c *Coloring) Init() int {
	c.bits = 0
	return readline.SGR3(22, 49, 39)
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
		return readline.SGR3(1, 49, 31) // red
	}
	return readline.SGR3(1, 49, 36) // cyan
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
			return fmt.Fprint(w, "SQL> ")
		}
		return fmt.Fprintf(w, "%3d> ", i+1)
	}
	for {
		lines, err := editor.Read(ctx)
		if err != nil {
			return err
		}
		sql := strings.Join(lines, "\n")
		sql = strings.TrimSpace(sql)
		if sql[len(sql)-1] == ';' {
			sql = sql[:len(sql)-1]
			sql = strings.TrimSpace(sql)
		}
		history.Add(sql)
		fields := strings.Fields(sql)
		if len(fields) <= 0 {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "SELECT":
			rows, err := conn.QueryContext(ctx, sql)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Query: %s\n", err.Error())
				continue
			}
			if err := dumpRows(ctx, rows, ",", "\n", os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			rows.Close()
		case "DELETE", "INSERT", "UPDATE":
			result, err := conn.ExecContext(ctx, sql)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Exec: %s\n", err.Error())
				continue
			}
			if count, err := result.RowsAffected(); err == nil {
				fmt.Fprintf(os.Stderr, "%d record(s) updated.\n", count)
			}
		case "EXIT", "QUIT":
			return io.EOF
		default:
			_, err := conn.ExecContext(ctx, sql)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Exec: %s\n", err.Error())
			}
			fmt.Fprintln(os.Stderr, "OK")
		}
	}
}

func mains(args []string) error {
	if len(args) <= 0 {
		return errors.New("Usage: sqlbless oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE")
	}
	fmt.Println("SQL*Bless")
	fmt.Println("  Ctrl-M or      Enter: Insert Linefeed")
	fmt.Println("  Ctrl-J or Ctrl-Enter: Exec command")
	fmt.Println()
	conn, err := sql.Open("oracle", args[0])
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
