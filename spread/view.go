package spread

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/hymkor/csvi"
	"github.com/hymkor/csvi/uncsv"
	"github.com/hymkor/sqlbless/rowstocsv"
)

type Viewer struct {
	HeaderLines int
	Comma       byte
	Null        string
	Spool       io.Writer
	csvi.Pilot
}

const (
	titlePrefix = "["
	titleSuffix = "]"
)

func (viewer *Viewer) View(ctx context.Context, title string, rows rowstocsv.Source, termOut io.Writer) error {
	cfg := &csvi.Config{
		Titles:   []string{toOneLine(title, titlePrefix, titleSuffix)},
		ReadOnly: true,
		Pilot:    viewer.Pilot,
	}
	csvWriteTo := func(w io.Writer) error {
		return rowstocsv.Config{
			Null:      viewer.Null,
			Comma:     rune(viewer.Comma),
			AutoClose: true,
		}.Dump(ctx, rows, w)
	}
	_, err := viewer.callCsvi(cfg, csvWriteTo, termOut)
	return err
}

func (viewer *Viewer) edit(title string, validate func(*csvi.CellValidatedEvent) (string, error), csvWriteTo func(pOut io.Writer) error, termOut io.Writer) (*csvi.Result, error) {

	applyChange := false
	setNull := func(e *csvi.KeyEventArgs) (*csvi.CommandResult, error) {
		if e.CursorRow.Index() < viewer.HeaderLines {
			return &csvi.CommandResult{}, nil
		}
		ce := &csvi.CellValidatedEvent{
			Text: viewer.Null,
			Row:  e.CursorRow.Index(),
			Col:  e.CursorCol,
		}
		if _, err := validate(ce); err != nil {
			return &csvi.CommandResult{Message: err.Error()}, nil
		}
		e.CursorRow.Replace(e.CursorCol, viewer.Null, &uncsv.Mode{Comma: viewer.Comma})
		return &csvi.CommandResult{}, nil
	}

	quit := func(app *csvi.KeyEventArgs) (*csvi.CommandResult, error) {
		ch, err := app.MessageAndGetKey(`"Y": Save&Exit  "N": Discard&Exit  <ESC>: Cancel(edit)`)
		if err != nil {
			return nil, err
		}
		switch ch {
		case "y", "Y":
			io.WriteString(app, " y\n")
			applyChange = true
			return &csvi.CommandResult{Quit: true}, nil
		case "n", "N":
			io.WriteString(app, " n\n")
			return &csvi.CommandResult{Quit: true}, nil
		default:
			return &csvi.CommandResult{}, nil
		}
	}

	apply := func(app *csvi.KeyEventArgs) (*csvi.CommandResult, error) {
		if app.YesNo("Apply changes and quit ? [y/n] ") {
			io.WriteString(app, "y\n")
			applyChange = true
			return &csvi.CommandResult{Quit: true}, nil
		}
		return &csvi.CommandResult{}, nil
	}

	cfg := &csvi.Config{
		Titles: []string{
			toOneLine(title, titlePrefix, titleSuffix),
			"ESC+\"y\": Apply changes & quit, ESC+\"n\": Discard changes & quit",
		},
		KeyMap: map[string]func(*csvi.KeyEventArgs) (*csvi.CommandResult, error){
			"\x1B": quit,
			"q":    quit,
			"c":    apply,
			"x":    setNull,
			"d":    setNull,
		},
		OnCellValidated: validate,
		Pilot:           viewer.Pilot,
	}
	result, err := viewer.callCsvi(cfg, csvWriteTo, termOut)
	if applyChange {
		return result, err
	}
	return nil, err
}

func toOneLine(s, prefix, suffix string) string {
	s = strings.TrimSpace(s)
	var buf strings.Builder
	buf.WriteString(prefix)
	var lastc rune
	quote := false
	for _, c := range s {
		if c <= ' ' {
			if lastc > ' ' || quote {
				buf.WriteRune(' ')
			}
		} else {
			buf.WriteRune(c)
		}
		if c == '\'' {
			quote = !quote
		}
		lastc = c
	}
	buf.WriteString(suffix)
	return buf.String()
}

func (viewer *Viewer) callCsvi(cfg *csvi.Config, csvWriteTo func(pOut io.Writer) error, termOut io.Writer) (*csvi.Result, error) {

	var err1 error
	pIn, pOut := io.Pipe()
	go func() {
		var w io.Writer
		if viewer.Spool != nil {
			w = io.MultiWriter(pOut, viewer.Spool)
		} else {
			w = pOut
		}
		err1 = csvWriteTo(w)
		pOut.Close()
	}()

	cfg.Mode = &uncsv.Mode{Comma: viewer.Comma}
	cfg.HeaderLines = viewer.HeaderLines
	cfg.FixColumn = true
	cfg.ProtectHeader = true

	result, err2 := cfg.Edit(pIn, termOut)
	pIn.Close()
	return result, errors.Join(err1, err2)
}
