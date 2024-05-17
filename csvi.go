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

func csvPager(title string, f func(pOut io.Writer) error, out, spool io.Writer) error {
	_, err := csvEdit(title, true, f, out, spool)
	return err
}

func csvEdit(title string, readonly bool, f func(pOut io.Writer) error, out, spool io.Writer) (result *csvi.Result, err error) {
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
	}
	if *flagAuto != "" {
		cfg.Pilot = _QuitCsvi{}
	}
	var err2 error
	result, err2 = cfg.Edit(pIn, out)
	err = errors.Join(err, err2)

	pIn.Close()
	return result, err
}
