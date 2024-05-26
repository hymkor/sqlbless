package main

import (
	"errors"
	"io"
	"strings"

	"github.com/hymkor/csvi"
	"github.com/hymkor/csvi/uncsv"
)

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

type getKeyAndSize interface {
	GetKey() (string, error)
	Size() (int, int, error)
}

type _AutoCsvi struct {
	Tty getKeyAndSize
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

func csvPager(title string, quitImmediate bool, f func(pOut io.Writer) error, out, spool io.Writer) error {
	var pilot csvi.Pilot
	if quitImmediate {
		pilot = _QuitCsvi{}
	}
	_, err := _csvEdit(title, true, pilot, f, out, spool)
	return err
}

func csvEdit(title string, tty getKeyAndSize, f func(pOut io.Writer) error, out, spool io.Writer) (result *csvi.Result, err error) {
	var pilot csvi.Pilot
	if tty != nil {
		pilot = &_AutoCsvi{Tty: tty}
	}
	return _csvEdit(title, false, pilot, f, out, spool)
}

func _csvEdit(title string, readonly bool, pilot csvi.Pilot, f func(pOut io.Writer) error, out, spool io.Writer) (result *csvi.Result, err error) {

	pIn, pOut := io.Pipe()
	go func() {
		if spool == nil {
			err = f(pOut)
		} else {
			err = f(tee(pOut, spool))
		}
		pOut.Close()
	}()

	var titleBuf strings.Builder
	for _, c := range title {
		if c < ' ' {
			titleBuf.WriteRune(0x2400 + c)
		} else {
			titleBuf.WriteRune(c)
		}
	}

	cfg := &csvi.Config{
		Mode:        &uncsv.Mode{Comma: ','},
		CellWidth:   14,
		HeaderLines: 1,
		FixColumn:   true,
		ReadOnly:    readonly,
		Message:     titleBuf.String(),
		Pilot:       pilot,
		KeyMap:      make(map[string]func(*csvi.Application) (*csvi.CommandResult, error)),
	}
	applyChange := false
	if !readonly {
		cfg.KeyMap["c"] = func(app *csvi.Application) (*csvi.CommandResult, error) {
			if app.YesNo("Apply the changes ? [y/n] ") {
				io.WriteString(app, "y\n")
				applyChange = true
				return &csvi.CommandResult{Quit: true}, nil
			}
			return &csvi.CommandResult{}, nil
		}
	}
	var err2 error
	result, err2 = cfg.Edit(pIn, out)
	err = errors.Join(err, err2)
	pIn.Close()
	if applyChange {
		return result, err
	}
	return nil, err
}
