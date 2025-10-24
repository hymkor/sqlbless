package misc

import (
	"io"
	"strings"

	"github.com/hymkor/csvi/candidate"
)

type AutoCsvi struct {
	GetKeyAndSize
}

func (ac AutoCsvi) readline() (string, error) {
	var buffer strings.Builder
	for {
		c, err := ac.GetKey()
		if err != nil {
			return "", err
		}
		if c == "\r" || c == "\n" {
			return buffer.String(), nil
		}
		buffer.WriteString(c)
	}
}

func (ac AutoCsvi) ReadLine(io.Writer, string, string, candidate.Candidate) (string, error) {
	return ac.readline()
}

func (ac AutoCsvi) GetFilename(io.Writer, string, string) (string, error) {
	return ac.readline()
}

func (AutoCsvi) Close() error {
	return nil
}
