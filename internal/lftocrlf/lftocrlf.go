package lftocrlf

import (
	"bytes"
	"io"
)

type WriteNameCloser interface {
	Write([]byte) (int, error)
	Name() string
	Close() error
}

type LfToCrlf struct {
	WriteNameCloser
	crIsCut bool
}

func write(w io.Writer, crIsCut bool, block []byte) (int, bool, error) {
	n := 0
	if crIsCut {
		block = append([]byte{'\r'}, block...)
		crIsCut = false
		n = -1
	}
	for {
		var line []byte
		var ok bool

		line, block, ok = bytes.Cut(block, []byte{'\n'})
		n += len(line)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
			if !ok {
				crIsCut = true
			}
		}
		if len(line) > 0 {
			_, err := w.Write(line)
			if err != nil {
				return n, crIsCut, err
			}
		}
		if !ok {
			return n, crIsCut, nil
		}
		_, err := w.Write([]byte{'\r', '\n'})
		if err != nil {
			return 0, crIsCut, err
		}
		n++
	}
}

func (W *LfToCrlf) Write(b []byte) (int, error) {
	var n int
	var err error
	n, W.crIsCut, err = write(W.WriteNameCloser, W.crIsCut, b)
	return n, err
}

func New(fd WriteNameCloser) *LfToCrlf {
	return &LfToCrlf{WriteNameCloser: fd}
}
