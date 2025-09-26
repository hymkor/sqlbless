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
	"github.com/hymkor/go-shellcommand"
	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/auto"
	"github.com/nyaosorg/go-readline-ny/keys"

	"github.com/hymkor/sqlbless/dialect"
	"github.com/hymkor/sqlbless/internal/history"
	"github.com/hymkor/sqlbless/internal/lftocrlf"
	"github.com/hymkor/sqlbless/internal/sqlcompletion"
	"github.com/hymkor/sqlbless/misc"
	"github.com/hymkor/sqlbless/rowstocsv"
)

func doSelect(ctx context.Context, ss *Session, query string) error {
	var rows *sql.Rows
	var err error
	if ss.tx != nil {
		rows, err = ss.tx.QueryContext(ctx, query)
	} else {
		rows, err = ss.conn.QueryContext(ctx, query)
	}
	if err != nil {
		return fmt.Errorf("query: %[1]w (%[1]T)", err)
	}
	_rows, ok := rowsHasNext(rows)
	if !ok {
		rows.Close()
		return fmt.Errorf("data not found")
	}
	return newViewer(ss).View(ctx, query, ss.automatic, _rows, ss.termOut)
}

type canExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func doDML(ctx context.Context, conn canExec, query string, args []any, w io.Writer) (int64, error) {
	result, err := conn.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("exec: %[1]w (%[1]T)", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("RowsAffected: %[1]w (%[1]T)", err)
	}
	fmt.Fprintf(w, "%d record(s) updated.\n", count)
	return count, nil
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

func echo(spool io.Writer, query string) {
	echoPrefix(spool, "", query)
}

func echoPrefix(spool io.Writer, prefix, query string) {
	if spool != nil {
		fmt.Fprintf(spool, "### <%s> ###\n", time.Now().Local().Format(time.DateTime))
		query = strings.TrimRight(query, "\n")
		for {
			var line string
			var next bool
			line, query, next = strings.Cut(query, "\n")
			fmt.Fprintf(spool, "# %s%s\n", prefix, line)
			if !next {
				break
			}
		}
	}
}

func (ss *Session) desc(ctx context.Context, table string) error {
	// fmt.Fprintln(os.Stderr, dbDialect.SqlForDesc)
	tableName := strings.TrimSpace(table)
	var rows *sql.Rows
	var err error
	var title string
	if tableName == "" {
		if ss.Dialect.SqlForTab == "" {
			return errors.New("DESC: not supported")
		}
		if ss.Debug {
			fmt.Println(ss.Dialect.SqlForTab)
		}
		title = "Tables"
		rows, err = ss.conn.QueryContext(ctx, ss.Dialect.SqlForTab)
	} else {
		if ss.Dialect.SqlForDesc == "" {
			return errors.New("DESC TABLE: not supported")
		}
		sql := strings.ReplaceAll(ss.Dialect.SqlForDesc, "{table_name}", tableName)
		if ss.Debug {
			fmt.Println(sql)
		}
		title = tableName
		rows, err = ss.conn.QueryContext(ctx, sql, tableName)
	}
	if err != nil {
		return err
	}
	_rows, ok := rowsHasNext(rows)
	if !ok {
		rows.Close()
		if table == "" {
			return errors.New("no tables are found")
		}
		return fmt.Errorf("%s: table not found", table)
	}
	return newViewer(ss).View(ctx, title, ss.automatic, _rows, ss.termOut)
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

var o = struct{}{}

var oneLineCommands = map[string]struct{}{
	`DESC`:    o,
	`EDIT`:    o,
	`EXIT`:    o,
	`HISTORY`: o,
	`HOST`:    o,
	`QUIT`:    o,
	`REM`:     o,
	`SPOOL`:   o,
	`START`:   o,
	`\D`:      o,
}

func isOneLineCommand(cmdLine string) bool {
	first, _ := misc.CutField(cmdLine)
	first = strings.ToUpper(first)
	first = strings.TrimRight(first, ";")
	_, ok := oneLineCommands[first]
	return ok
}

type CommandIn interface {
	Read(context.Context) ([]string, error)
	GetKey() (string, error)

	// AutoPilotForCsvi returns Tty object when AutoPilot is enabled.
	// When disabled, it MUST return nil.
	AutoPilotForCsvi() (getKeyAndSize, bool)
}

type Script struct {
	br   *bufio.Reader
	echo io.Writer
	term string
}

func (script *Script) GetKey() (string, error) {
	return "", io.EOF
}

func (script *Script) AutoPilotForCsvi() (getKeyAndSize, bool) {
	return nil, false
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

func (i *InteractiveIn) AutoPilotForCsvi() (getKeyAndSize, bool) {
	return i.tty, (i.tty != nil)
}

type Session struct {
	rowstocsv.Config
	Dialect         *dialect.Entry
	conn            *sql.DB
	history         *history.History
	tx              *sql.Tx
	spool           lftocrlf.WriteNameCloser
	stdOut, termOut io.Writer
	stdErr, termErr io.Writer
	automatic       bool
	term            string
	crlf            bool
}

func (ss *Session) Close() {
	if ss.tx != nil {
		txRollback(&ss.tx, ss.stdErr)
	}
	if ss.spool != nil {
		ss.spool.Close()
		ss.spool = nil
		ss.stdOut = ss.termOut
		ss.stdErr = ss.termErr
	}
}

func (ss *Session) StartFromStdin(ctx context.Context) error {
	script := &Script{
		br:   bufio.NewReader(os.Stdin),
		echo: ss.stdErr,
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
		echo: ss.stdErr,
		term: ss.term,
	}
	return ss.Loop(ctx, script, true)
}

func (ss *Session) Loop(ctx context.Context, commandIn CommandIn, onErrorAbort bool) error {
	for {
		if ss.spool != nil {
			fmt.Fprintf(ss.termErr, "\nSpooling to '%s' now\n", ss.spool.Name())
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
		cmd, arg := misc.CutField(query)
		switch strings.ToUpper(cmd) {
		case "REM":
			// nothing to do
		case "HOST":
			process, err := shellcommand.System(arg)
			if err == nil {
				process.Wait()
			}
		case "SPOOL":
			fname, _ := misc.CutField(arg)
			if fname == "" {
				if ss.spool != nil {
					fmt.Fprintf(ss.termErr, "Spooling to '%s' now\n", ss.spool.Name())
				} else {
					fmt.Fprintln(ss.termErr, "Not Spooling")
				}
				continue
			}
			if ss.spool != nil {
				ss.spool.Close()
				fmt.Fprintln(ss.termErr, "Spool closed.")
				ss.spool = nil
				ss.stdOut = ss.termOut
				ss.stdErr = ss.termErr
			}
			if !strings.EqualFold(fname, "off") {
				if fd, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					if ss.crlf {
						ss.spool = lftocrlf.New(fd)
					} else {
						ss.spool = fd
					}
					ss.stdOut = io.MultiWriter(ss.termOut, ss.spool)
					ss.stdErr = io.MultiWriter(ss.termErr, ss.spool)
					fmt.Fprintf(ss.termErr, "Spool to %s\n", fname)
					writeSignature(ss.spool)
				}
			}
		case "EDIT":
			echo(ss.spool, query)
			err = doEdit(ctx, ss, query, commandIn)

		case "SELECT":
			echo(ss.spool, query)
			err = doSelect(ctx, ss, query)
		case "DELETE", "INSERT", "UPDATE":
			echo(ss.spool, query)
			isNewTx := (ss.tx == nil)
			err = txBegin(ctx, ss.conn, &ss.tx, ss.stdErr)
			if err == nil {
				count, err := doDML(ctx, ss.tx, query, nil, ss.stdOut)
				if (err != nil || count == 0) && isNewTx && ss.tx != nil {
					ss.tx.Rollback()
					ss.tx = nil
				}
			}
		case "COMMIT":
			echo(ss.spool, query)
			err = txCommit(&ss.tx, ss.stdErr)
		case "ROLLBACK":
			echo(ss.spool, query)
			err = txRollback(&ss.tx, ss.stdErr)
		case "EXIT", "QUIT":
			return nil
		case "DESC", "\\D":
			echo(ss.spool, query)
			err = ss.desc(ctx, arg)
		case "HISTORY":
			echo(ss.spool, query)
			csvw := csv.NewWriter(ss.stdOut)
			for i, end := 0, ss.history.Len(); i < end; i++ {
				text, stamp := ss.history.TextAndStamp(i)
				csvw.Write([]string{
					strconv.Itoa(i),
					stamp.Local().Format(time.DateTime),
					text})
			}
			csvw.Flush()
		case "START":
			fname, _ := misc.CutField(arg)
			err = ss.Start(ctx, fname)
		default:
			echo(ss.spool, query)
			if ss.tx != nil {
				err = errors.New("transaction is not closed. Please Commit or Rollback")
			} else {
				_, err = ss.conn.ExecContext(ctx, query)
				if err == nil {
					fmt.Fprintln(ss.stdErr, "Ok")
				}
			}
		}
		if err != nil {
			fmt.Fprintln(ss.stdErr, err.Error())
			if onErrorAbort {
				return err
			}
		}
	}
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
	SpoolFilename  string
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

func (cfg Config) Run(driver, dataSourceName string, dbDialect *dialect.Entry) error {
	disabler := colorable.EnableColorsStdout(nil)
	defer disabler()
	termOut := colorable.NewColorableStdout()
	termErr := colorable.NewColorableStderr()

	conn, err := sql.Open(driver, dataSourceName)
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer conn.Close()

	if err = conn.Ping(); err != nil {
		return err
	}

	var spool lftocrlf.WriteNameCloser
	if fn := cfg.SpoolFilename; fn != "" &&
		!strings.EqualFold(fn, os.DevNull) &&
		!strings.EqualFold(fn, "off") {

		fd, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		} else {
			spool = fd
		}
	}

	var history history.History

	session := &Session{
		Dialect:   dbDialect,
		conn:      conn,
		history:   &history,
		automatic: cfg.Auto != "",
		term:      cfg.Term,
		crlf:      cfg.CrLf,
		stdOut:    termOut,
		termOut:   termOut,
		stdErr:    termErr,
		termErr:   termErr,
		spool:     spool,
	}
	defer session.Close()

	session.Null = cfg.Null
	if cfg.Tsv {
		session.Comma = '\t'
	} else {
		session.Comma, _ = utf8.DecodeRuneInString(cfg.FieldSeperator)
	}
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
	editor.SetWriter(termOut)

	editor.BindKey(keys.CtrlI, &completion.CmdCompletionOrList{
		Enclosure:  `"'`,
		Delimiter:  ",",
		Postfix:    " ",
		Candidates: sqlcompletion.New(dbDialect, conn),
	})
	editor.SubmitOnEnterWhen(func(lines []string, csrline int) bool {
		if len(lines) > 0 && isOneLineCommand(lines[0]) {
			return true
		}
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
