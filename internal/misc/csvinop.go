package misc

import (
	"io"

	"github.com/hymkor/csvi/candidate"
)

type GetKeyAndSize interface {
	GetKey() (string, error)
	Size() (int, int, error)
}

type CsviNoOperation struct {
	text []string
}

func (*CsviNoOperation) Size() (int, int, error) {
	return 80, 25, nil
}

func (c *CsviNoOperation) GetKey() (string, error) {
	if len(c.text) <= 0 {
		c.text = []string{">", "q", "y", ""}
	}
	v := c.text[0]
	if v == "" {
		return "", io.EOF
	}
	c.text = c.text[1:]
	return v, nil
}

func (*CsviNoOperation) ReadLine(io.Writer, string, string, candidate.Candidate) (string, error) {
	return "", nil
}

func (*CsviNoOperation) GetFilename(io.Writer, string, string) (string, error) {
	return "", nil
}

func (*CsviNoOperation) Close() error {
	return nil
}
