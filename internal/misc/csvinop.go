package misc

import (
	"io"

	"github.com/hymkor/csvi/candidate"
)

type GetKeyAndSize interface {
	GetKey() (string, error)
	Size() (int, int, error)
}

type CsviNoOperation struct{}

func (CsviNoOperation) Size() (int, int, error) {
	return 80, 25, nil
}

func (CsviNoOperation) Calibrate() error {
	return nil
}

func (CsviNoOperation) GetKey() (string, error) {
	return "q", nil
}

func (CsviNoOperation) ReadLine(io.Writer, string, string, candidate.Candidate) (string, error) {
	return "", nil
}

func (CsviNoOperation) GetFilename(io.Writer, string, string) (string, error) {
	return "", nil
}

func (CsviNoOperation) Close() error {
	return nil
}
