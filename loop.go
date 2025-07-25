package sqlbless

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-tty"

	"github.com/hymkor/go-multiline-ny"
	"github.com/hymkor/go-multiline-ny/completion"
	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/auto"
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

func doSelect(ctx context.Context, ss *Session, query string, out io.Writer) error {
	var conn canQuery
	if ss.tx == nil {
		conn = ss.conn
	} else {
		conn = ss.tx
	}
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query: %[1]w (%[1]T)", err)
	}
	_rows := rowsHasNext(rows)
	if _rows == nil {
		rows.Close()
		return fmt.Errorf("data not found")
	}
	return csvPager(query, ss, ss.automatic, func(pOut io.Writer) error {
		_err := ss.DumpConfig.Dump(ctx, _rows, pOut)
		rows.Close()
		return _err
	}, out)
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
	// fmt.Fprintln(os.Stderr, dbDialect.SqlForDesc)
	tableName := strings.TrimSpace(table)
	var rows *sql.Rows
	var err error
	if tableName == "" {
		if ss.dbDialect.SqlForTab == "" {
			return errors.New("DESC: not supported")
		}
		if ss.DumpConfig.PrintType {
			fmt.Println(ss.dbDialect.SqlForTab)
		}
		rows, err = ss.conn.QueryContext(ctx, ss.dbDialect.SqlForTab)
	} else {
		if ss.dbDialect.SqlForDesc == "" {
			return errors.New("DESC TABLE: not supported")
		}
		sql := strings.ReplaceAll(ss.dbDialect.SqlForDesc, "{table_name}", tableName)
		if ss.DumpConfig.PrintType {
			fmt.Println(sql)
		}
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
	return csvPager(table, ss, ss.automatic, func(pOut io.Writer) error {
		err := ss.DumpConfig.Dump(ctx, _rows, pOut)
		rows.Close()
		return err
	}, out)
}

// hasTerm is similar with strings.HasSuffix, but ignores cases when comparing and returns the trimed string and the boolean indicating trimed or not
func hasTerm(s, term string) (string, bool) {
	s = strings.TrimRight(s, " \r\n\t\v")
	from := len(s) - len(term)
	if 0 <= from && from < len(s) && strings.EqualFold(s[from:], term) {
		return s[:from], true
	}
	return s, false
}

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
	term string
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
		if ch == '\r' {
			continue
		} else if ch == '\'' {
			quoted ^= 1
		} else if ch == '"' {
			quoted ^= 2
		}
		buffer.WriteRune(ch)

		if quoted == 0 {
			code := buffer.String()
			term := script.term
			if _, ok := hasTerm(code, term); ok {
				println(code)
				fmt.Fprintln(script.echo, code)
				return []string{code}, nil
			}
		}
	}
}

type InteractiveIn struct {
	*multiline.Editor
	tty getKeyAndSize
}

func (i *InteractiveIn) GetKey() (string, error) {
	if i.tty != nil {
		return i.tty.GetKey()
	}
	tt, err := tty.Open()
	if err != nil {
		return "", err
	}
	defer tt.Close()
	return readline.GetKey(tt)
}

func (i *InteractiveIn) AutoPilotForCsvi() getKeyAndSize {
	return i.tty
}

type Session struct {
	DumpConfig RowToCsv
	dbDialect  *DBDialect
	conn       *sql.DB
	history    *History
	tx         *sql.Tx
	spool      FilterSource
	automatic  bool
	term       string
	crlf       bool
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
		term: ss.term,
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
		term: ss.term,
	}
	return ss.Loop(ctx, script, true)
}

func (ss *Session) Loop(ctx context.Context, commandIn CommandIn, onErrorAbort bool) error {
	for {
		if ss.spool != nil {
			fmt.Fprintf(os.Stderr, "\nSpooling to '%s' now\n", ss.spool.Name())
		}
		type PromptSetter interface {
			SetPrompt(func(io.Writer, int) (int, error))
		}
		if ps, ok := commandIn.(PromptSetter); ok {
			ps.SetPrompt(func(w io.Writer, i int) (int, error) {
				io.WriteString(w, "\x1B[0m")
				if i <= 0 {
					if ss.tx != nil {
						return io.WriteString(w, "SQL* ")
					}
					return io.WriteString(w, "SQL> ")
				}
				if ss.tx != nil {
					return fmt.Fprintf(w, "%3d* ", i+1)
				}
				return fmt.Fprintf(w, "%3d> ", i+1)
			})
		}
		lines, err := commandIn.Read(ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if in, ok := commandIn.(*InteractiveIn); ok {
			if L := len(lines) - 1; L > 0 {
				w := in.LineEditor.Out
				fmt.Fprintf(w, "\x1B[%dF", L)
				for ; L > 0; L-- {
					fmt.Fprintln(w, "     ")
				}
				w.Flush()
			}
		}
		queryAndTerm := strings.Join(lines, "\n")
		query, _ := hasTerm(queryAndTerm, ss.term)

		if query == "" {
			continue
		}
		if _, ok := commandIn.(*Script); !ok {
			ss.history.Add(queryAndTerm)
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
					if ss.crlf {
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
			err = doEdit(ctx, ss, query, commandIn, os.Stdout)

		case "SELECT":
			echo(ss.spool, query)
			err = doSelect(ctx, ss, query, os.Stdout)
		case "DELETE", "INSERT", "UPDATE":
			echo(ss.spool, query)
			isNewTx := (ss.tx == nil)
			err = txBegin(ctx, ss.conn, &ss.tx, tee(os.Stderr, ss.spool))
			if err == nil {
				err = doDML(ctx, ss.tx, query, tee(os.Stdout, ss.spool))
				if err != nil && isNewTx && ss.tx != nil {
					ss.tx.Rollback()
					ss.tx = nil
				}
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

func sqlCandidates(field []string) (fullnames []string, basenames []string) {
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
	basenames = fullnames
	return
}

type Config struct {
	Auto           string
	Term           string
	CrLf           bool
	Null           string
	Tsv            bool
	FieldSeperator string
	Debug          bool
	SubmitByEnter  bool
	Script         string
}

type ReservedWordPattern map[string]struct{}

var rxWords = regexp.MustCompile(`\b\w+\b`)

func (h ReservedWordPattern) FindAllStringIndex(s string, n int) [][]int {
	matches := rxWords.FindAllStringIndex(s, n)
	for i := len(matches) - 1; i >= 0; i-- {
		word := s[matches[i][0]:matches[i][1]]
		if _, ok := h[strings.ToUpper(word)]; !ok {
			copy(matches[i:], matches[i+1:])
			matches = matches[:len(matches)-1]
		}
	}
	return matches
}

func newReservedWordPattern(list ...string) ReservedWordPattern {
	m := ReservedWordPattern{}
	for _, word := range list {
		m[strings.ToUpper(word)] = struct{}{}
	}
	return m
}

func (cfg Config) Run(driver, dataSourceName string, dbDialect *DBDialect) error {
	conn, err := sql.Open(driver, dataSourceName)
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer conn.Close()

	if err = conn.Ping(); err != nil {
		return err
	}

	var history History

	session := &Session{
		dbDialect: dbDialect,
		conn:      conn,
		history:   &history,
		automatic: cfg.Auto != "",
		term:      cfg.Term,
		crlf:      cfg.CrLf,
	}
	defer session.Close()

	session.DumpConfig.Null = cfg.Null
	if cfg.Tsv {
		session.DumpConfig.Comma = '\t'
	} else {
		session.DumpConfig.Comma, _ = utf8.DecodeRuneInString(cfg.FieldSeperator)
	}
	session.DumpConfig.TimeLayout = dbDialect.DisplayDateTimeLayout
	session.DumpConfig.PrintType = cfg.Debug

	ctx := context.Background()
	if cfg.Script != "" {
		return session.Start(ctx, cfg.Script)
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

	editor.ResetColor = "\x1B[0m"
	editor.DefaultColor = "\x1B[39;49;1m"
	editor.Highlight = []readline.Highlight{
		{Pattern: newReservedWordPattern("ALTER", "COMMIT", "CREATE", "DELETE", "DESC", "DROP", "EXIT", "HISTORY", "INSERT", "QUIT", "REM", "ROLLBACK", "SELECT", "SPOOL", "START", "TRUNCATE", "UPDATE", "AND", "FROM", "INTO", "OR", "WHERE"), Sequence: "\x1B[36;49;1m"},
		{Pattern: regexp.MustCompile(`[0-9]+`), Sequence: "\x1B[35;49;1m"},
		{Pattern: regexp.MustCompile(`/\*.*?\*/`), Sequence: "\x1B[33;49;22m"},
		{Pattern: regexp.MustCompile(`"[^"]*"|"[^"]*$`), Sequence: "\x1B[31;49;1m"},
		{Pattern: regexp.MustCompile(`'[^']*'|'[^']*$`), Sequence: "\x1B[35;49;1m"},
	}

	if cfg.SubmitByEnter {
		editor.SwapEnter()
	}
	var tty getKeyAndSize
	if cfg.Auto != "" {
		text := strings.ReplaceAll(cfg.Auto, "||", "\n") // "||" -> Ctrl-J(Commit)
		text = strings.ReplaceAll(text, "|", "\r")       // "|" -> Ctrl-M (NewLine)
		if text[len(text)-1] != '\n' {                   // EOF -> Ctrl-J(Commit)
			text = text + "\n"
		}
		tty1 := &auto.Pilot{
			Text: strings.Split(text, ""),
		}
		editor.LineEditor.Tty = tty1
		tty = tty1
	}
	editor.SetPredictColor(readline.PredictColorBlueItalic)
	editor.SetHistory(&history)
	editor.SetWriter(colorable.NewColorableStdout())

	completer := &completeType{
		conn:        conn,
		SqlForTab:   dbDialect.SqlForTab,
		SqlForDesc:  dbDialect.SqlForDesc,
		TableField:  dbDialect.TableField,
		ColumnField: dbDialect.ColumnField,
	}
	editor.BindKey(keys.CtrlI, &completion.CmdCompletionOrList{
		Enclosure:  `"'`,
		Delimiter:  ",",
		Postfix:    " ",
		Candidates: completer.getCandidates,
	})
	editor.SubmitOnEnterWhen(func(lines []string, csrline int) bool {
		for {
			last := strings.TrimRight(lines[len(lines)-1], " \r\n\t\v")
			if last != "" || len(lines) <= 1 {
				if len(cfg.Term) == 1 {
					_, ok := hasTerm(last, cfg.Term)
					return ok
				} else {
					return strings.EqualFold(last, cfg.Term)
				}
			}
			lines = lines[:len(lines)-1]
		}
	})

	return session.Loop(ctx, &InteractiveIn{
		Editor: &editor,
		tty:    tty,
	}, false)
}
