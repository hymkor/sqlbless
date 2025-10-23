package sqlbless

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/mattn/go-colorable"

	"github.com/hymkor/csvi"
	"github.com/hymkor/go-shellcommand"

	"github.com/hymkor/sqlbless/dialect"

	"github.com/hymkor/sqlbless/internal/history"
	"github.com/hymkor/sqlbless/internal/lftocrlf"
	"github.com/hymkor/sqlbless/internal/misc"
)

type commandIn interface {
	Read(context.Context) ([]string, error)
	GetKey() (string, error)

	// AutoPilotForCsvi returns Tty object when AutoPilot is enabled.
	// When disabled, it MUST return nil.
	AutoPilotForCsvi() (csvi.Pilot, bool)

	CanCloseInTransaction() bool
	ShouldRecordHistory() bool
	SetPrompt(func(io.Writer, int) (int, error))
}

type session struct {
	*Config
	Dialect         *dialect.Entry
	conn            *sql.Conn
	history         *history.History
	tx              *sql.Tx
	spool           lftocrlf.WriteNameCloser
	stdOut, termOut io.Writer
	stdErr, termErr io.Writer
}

func (ss *session) Close() {
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

func (ss *session) automatic() bool {
	return ss.Auto != ""
}

var (
	ErrTransactionIsNotClosed = errors.New("transaction is not closed. Please Commit or Rollback")
	ErrBeginIsNotSupported    = errors.New("'BEGIN' is not supported; transactions are managed automatically")
)

func (ss *session) Loop(ctx context.Context, commandIn commandIn, onErrorAbort bool) error {
	for {
		if ss.spool != nil {
			fmt.Fprintf(ss.termErr, "\nSpooling to '%s' now\n", ss.spool.Name())
		}
		commandIn.SetPrompt(func(w io.Writer, i int) (int, error) {
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
		lines, err := commandIn.Read(ctx)
		if err != nil {
			if err == io.EOF {
				if ss.tx != nil && !commandIn.CanCloseInTransaction() {
					fmt.Fprintln(ss.termErr, ErrTransactionIsNotClosed.Error())
					continue
				}
				return nil
			}
			return err
		}
		queryAndTerm := strings.Join(lines, "\n")
		query, _ := misc.HasTerm(queryAndTerm, ss.Term)

		if query == "" {
			continue
		}
		if commandIn.ShouldRecordHistory() {
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
					if ss.CrLf {
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
			misc.Echo(ss.spool, query)
			err = doEdit(ctx, ss, query, commandIn)

		case "SELECT":
			misc.Echo(ss.spool, query)
			err = doSelect(ctx, ss, query)
		case "DELETE", "INSERT", "UPDATE":
			misc.Echo(ss.spool, query)
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
			misc.Echo(ss.spool, query)
			err = txCommit(&ss.tx, ss.stdErr)
		case "ROLLBACK":
			misc.Echo(ss.spool, query)
			err = txRollback(&ss.tx, ss.stdErr)
		case "EXIT", "QUIT":
			if ss.tx == nil || commandIn.CanCloseInTransaction() {
				return nil
			}
			err = ErrTransactionIsNotClosed
		case "DESC", "\\D":
			misc.Echo(ss.spool, query)
			err = ss.desc(ctx, arg)
		case "HISTORY":
			misc.Echo(ss.spool, query)
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
		case "BEGIN":
			err = ErrBeginIsNotSupported
		default:
			misc.Echo(ss.spool, query)
			if q := ss.Dialect.IsQuerySQL; q != nil && q(query) {
				err = doSelect(ctx, ss, query)
			} else {
				if ss.tx == nil {
					_, err = ss.conn.ExecContext(ctx, query)
				} else if f := ss.Dialect.CanUseInTransaction; f != nil && f(query) {
					_, err = ss.tx.ExecContext(ctx, query)
				} else {
					err = ErrTransactionIsNotClosed
				}
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

func (cfg *Config) openSpool() lftocrlf.WriteNameCloser {
	fn := cfg.SpoolFilename
	if fn == "" || strings.EqualFold(fn, os.DevNull) || strings.EqualFold(fn, "off") {
		return nil
	}
	fd, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil
	}
	return fd
}

func (cfg *Config) Run(driver, dataSourceName string, dbDialect *dialect.Entry) error {
	ctx := context.Background()

	if cfg.ReverseVideo || csvi.IsRevertVideoWithEnv() {
		csvi.RevertColor()
	}
	if noColor := os.Getenv("NO_COLOR"); len(noColor) > 0 {
		csvi.MonoChrome()
	}

	disabler := colorable.EnableColorsStdout(nil)
	defer disabler()
	termOut := colorable.NewColorableStdout()
	termErr := colorable.NewColorableStderr()

	db, err := sql.Open(driver, dataSourceName)
	if err != nil {
		return fmt.Errorf("sql.Open: %[1]w (%[1]T)", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ss := &session{
		Config:  cfg,
		Dialect: dbDialect,
		conn:    conn,
		history: &history.History{},
		stdOut:  termOut,
		termOut: termOut,
		stdErr:  termErr,
		termErr: termErr,
		spool:   cfg.openSpool(),
	}
	defer ss.Close()

	if cfg.Script != "" {
		return ss.Start(ctx, cfg.Script)
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return ss.StartFromStdin(ctx)
	}

	return ss.Loop(ctx, ss.newInteractiveIn(), false)
}
