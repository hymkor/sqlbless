package spread

import (
	"errors"
	"io"
	"strings"

	"github.com/hymkor/csvi"
	"github.com/hymkor/csvi/uncsv"
)

type Viewer struct {
	HeaderLines int
	Comma       byte
	Null        string
	Spool       io.Writer
}

type _QuitCsvi struct{}

func (_QuitCsvi) Size() (int, int, error) {
	return 80, 25, nil
}

func (_QuitCsvi) Calibrate() error {
	return nil
}

func (_QuitCsvi) GetKey() (string, error) {
	return "q", nil
}

func (_QuitCsvi) ReadLine(io.Writer, string, string, csvi.Candidate) (string, error) {
	return "", nil
}

func (_QuitCsvi) GetFilename(io.Writer, string, string) (string, error) {
	return "", nil
}

func (_QuitCsvi) Close() error {
	return nil
}

type GetKeyAndSize interface {
	GetKey() (string, error)
	Size() (int, int, error)
}

type _AutoCsvi struct {
	Tty GetKeyAndSize
}

func (_AutoCsvi) Calibrate() error {
	return nil
}

func (ac _AutoCsvi) Size() (int, int, error) {
	return ac.Tty.Size()
}

func (ac _AutoCsvi) GetKey() (string, error) {
	return ac.Tty.GetKey()
}

func (ac _AutoCsvi) readline() (string, error) {
	var buffer strings.Builder
	for {
		c, err := ac.Tty.GetKey()
		if err != nil {
			return "", err
		}
		if c == "\r" || c == "\n" {
			return buffer.String(), nil
		}
		buffer.WriteString(c)
	}
}

func (ac _AutoCsvi) ReadLine(io.Writer, string, string, csvi.Candidate) (string, error) {
	return ac.readline()
}

func (ac _AutoCsvi) GetFilename(io.Writer, string, string) (string, error) {
	return ac.readline()
}

func (_AutoCsvi) Close() error {
	return nil
}

const (
	titlePrefix = "【"
	titleSuffix = "】"
)

func (ss *Viewer) View(title string, automatic bool, csvWriteTo func(pOut io.Writer) error, out io.Writer) error {
	cfg := &csvi.Config{
		Titles:   []string{toOneLine(title, titlePrefix, titleSuffix)},
		ReadOnly: true,
	}
	if automatic {
		cfg.Pilot = _QuitCsvi{}
	}
	_, err := ss.callCsvi(cfg, csvWriteTo, out)
	return err
}

func (viewer *Viewer) edit(title string, validate func(*csvi.CellValidatedEvent) (string, error), tty GetKeyAndSize, csvWriteTo func(pOut io.Writer) error, out io.Writer) (*csvi.Result, error) {

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

	cfg := &csvi.Config{
		Titles: []string{
			toOneLine(title, titlePrefix, titleSuffix),
			"\"c\": Apply changes & quit, \"q\": Discard changes & quit",
		},
		KeyMap: map[string]func(*csvi.KeyEventArgs) (*csvi.CommandResult, error){
			"c": func(app *csvi.KeyEventArgs) (*csvi.CommandResult, error) {
				if app.YesNo("Apply changes and quit ? [y/n] ") {
					io.WriteString(app, "y\n")
					applyChange = true
					return &csvi.CommandResult{Quit: true}, nil
				}
				return &csvi.CommandResult{}, nil
			},
			"x": setNull,
			"d": setNull,
		},
		OnCellValidated: validate,
	}
	if tty != nil {
		cfg.Pilot = &_AutoCsvi{Tty: tty}
	}
	result, err := viewer.callCsvi(cfg, csvWriteTo, out)
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

func (viewer *Viewer) callCsvi(cfg *csvi.Config, csvWriteTo func(pOut io.Writer) error, out io.Writer) (*csvi.Result, error) {

	var err1 error
	pIn, pOut := io.Pipe()
	go func() {
		err1 = csvWriteTo(tee(pOut, viewer.Spool))
		pOut.Close()
	}()

	cfg.Mode = &uncsv.Mode{Comma: viewer.Comma}
	cfg.HeaderLines = viewer.HeaderLines
	cfg.FixColumn = true
	cfg.ProtectHeader = true

	result, err2 := cfg.Edit(pIn, out)
	pIn.Close()
	return result, errors.Join(err1, err2)
}
