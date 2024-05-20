package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-tty"

	"github.com/nyaosorg/go-readline-ny"

	"github.com/hymkor/csvi/uncsv"
)

type typeModified int

const (
	notModified = iota
	modified
	newRow
)

func csvRowModified(csvRow *uncsv.Row) typeModified {
	bits := 0
	for _, cell := range csvRow.Cell {
		if cell.Modified() {
			bits |= 1
		}
		if len(cell.Original()) > 0 {
			bits |= 2
		}
	}
	switch bits {
	case 1:
		return newRow
	case 3:
		return modified
	}
	return notModified
}

func askSqlAndExecute(ctx context.Context, ss *Session, dmlSql string) error {
	fmt.Println(dmlSql)
	tty1, err := tty.Open()
	if err != nil {
		return err
	}
	defer tty1.Close()
	fmt.Print("Execute? [y/n]")
	var answer string
	answer, err = readline.GetKey(tty1)
	fmt.Println()
	if err != nil {
		return err
	}
	if answer == "y" || answer == "Y" {
		err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
		if err != nil {
			return err
		}
		err = doDML(ctx, ss.tx, dmlSql, tee(os.Stdout, ss.spool))
		if err != nil {
			return err
		}
	}
	return nil
}

func createWhere(row *uncsv.Row, columns []string, columnQuotes [][2]string, null string) string {
	var where strings.Builder
	for i, c := range row.Cell {
		if i > 0 {
			where.WriteString("\n   AND  ")
		} else {
			where.WriteString("\n WHERE  ")
		}
		q := columnQuotes[i]
		if string(c.Original()) == null {
			fmt.Fprintf(&where, "%s is NULL", columns[i])
		} else {
			fmt.Fprintf(&where, "%s = %s%s%s", columns[i], q[0], c.Original(), q[1])
		}
	}
	return where.String()
}

func doEdit(ctx context.Context, ss *Session, command string, out, spool io.Writer) error {
	var conn canQuery
	if ss.tx == nil {
		conn = ss.conn
	} else {
		conn = ss.tx
	}

	// replace `edit ` to `select * from `
	_, tableAndWhere := cutField(command)
	query := "SELECT * FROM " + tableAndWhere

	table, _ := cutField(tableAndWhere)

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query: %[1]w (%[1]T)", err)
	}
	_rows := rowsHasNext(rows)
	if _rows == nil {
		rows.Close()
		return fmt.Errorf("data not found")
	}
	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return err
	}
	columnTypes, err := rows.ColumnTypes()
	columnQuotes := [][2]string{}
	for _, ct := range columnTypes {
		name := strings.ToUpper(ct.DatabaseTypeName())
		if strings.Contains(name, "INT") ||
			strings.Contains(name, "NUMBER") ||
			strings.Contains(name, "DECIMAL") {
			columnQuotes = append(columnQuotes, [2]string{"", ""})
		} else {
			columnQuotes = append(columnQuotes, [2]string{"'", "'"})
		}
	}
	editResult, err := csvEdit(command, false, func(pOut io.Writer) error {
		_err := ss.DumpConfig.Dump(ctx, _rows, pOut)
		rows.Close()
		return _err
	}, out, spool)

	if err != nil && err != io.EOF {
		return err
	}
	null := ss.DumpConfig.Null
	editResult.Each(func(row *uncsv.Row) bool {
		var dmlSql string
		switch csvRowModified(row) {
		case notModified:
			return true
		case newRow:
			var sql strings.Builder
			fmt.Fprintf(&sql, "INSERT INTO %s VALUES\n( ", table)
			for i, c := range row.Cell {
				if i > 0 {
					sql.WriteByte(',')
				}
				if c.Text() == null {
					sql.WriteString("NULL")
				} else {
					q := columnQuotes[i]
					fmt.Fprintf(&sql, "%s%s%s", q[0], c.Text(), q[1])
				}
			}
			sql.WriteString(")\n")
			dmlSql = sql.String()
		case modified:
			var sql strings.Builder
			sql.WriteString("UPDATE  ")
			sql.WriteString(table)

			del := "\n   SET  "

			for i, c := range row.Cell {
				q := columnQuotes[i]
				if c.Modified() {
					if c.Text() == null {
						fmt.Fprintf(&sql, "%s%s = NULL ",
							del,
							columns[i])
					} else {
						fmt.Fprintf(&sql, "%s%s = %s%s%s ",
							del,
							columns[i],
							q[0],
							c.Text(),
							q[1])
					}
					del = ",\n        "
				}
			}
			sql.WriteString(createWhere(row, columns, columnQuotes, null))
			dmlSql = sql.String()
		}
		err = askSqlAndExecute(ctx, ss, dmlSql)
		return err == nil
	})
	if err != nil {
		return err
	}
	editResult.RemovedRows(func(row *uncsv.Row) bool {
		var sql strings.Builder
		fmt.Fprintf(&sql, "DELETE FROM %s\n", table)
		sql.WriteString(createWhere(row, columns, columnQuotes, null))
		err = askSqlAndExecute(ctx, ss, sql.String())
		return err == nil
	})
	return err
}
