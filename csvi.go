package main

import (
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

func csvPager(title string, f func(pOut io.Writer) error, out io.Writer) (err error) {
	pIn, pOut := io.Pipe()
	go func() {
		err = f(pOut)
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
		CellWidth:   14,
		HeaderLines: 1,
		FixColumn:   true,
		ReadOnly:    true,
		Message:     titleBuf.String(),
	}
	if *flagAuto != "" {
		cfg.Pilot = _QuitCsvi{}
	}
	cfg.Main(&uncsv.Mode{Comma: ','}, pIn, out)

	pIn.Close()
	return
}
