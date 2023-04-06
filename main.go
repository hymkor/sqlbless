package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hymkor/go-multiline-ny"
	"github.com/nyaosorg/go-readline-ny/simplehistory"
	_ "github.com/sijms/go-ora/v2"
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

func loop(ctx context.Context, conn *sql.DB) error {
	var editor multiline.Editor
	history := simplehistory.New()
	editor.LineEditor.History = history

	editor.Prompt = func(w io.Writer, i int) (int, error) {
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
		history.Add(sql)
		rows, err := conn.QueryContext(ctx, sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Query: %s\n", err.Error())
			continue
		}
		if err := dumpRows(ctx, rows, ",", "\n", os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		rows.Close()
	}
}

func mains(args []string) error {
	if len(args) <= 0 {
		return errors.New("Usage: sqlbless oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE")
	}
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
