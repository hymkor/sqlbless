package sqlbless

import (
	"time"

	"github.com/nyaosorg/go-readline-ny"
)

type Coloring struct {
	bits int
}

func (c *Coloring) Init() readline.ColorSequence {
	c.bits = 0
	return readline.SGR1(0)
}

func (c *Coloring) Next(r rune) readline.ColorSequence {
	const (
		_SINGLE_QUOTED = 1
		_DOUBLE_QUOTED = 2
	)
	newbits := c.bits
	if r == '\'' {
		newbits ^= _SINGLE_QUOTED
	} else if r == '"' {
		newbits ^= _DOUBLE_QUOTED
	}
	defer func() {
		c.bits = newbits
	}()

	or := c.bits | newbits

	if (or & _SINGLE_QUOTED) != 0 {
		return readline.Red
	} else if (or & _DOUBLE_QUOTED) != 0 {
		return readline.Magenta
	}
	return readline.Cyan
}

type _HistoryLine struct {
	text  string
	stamp time.Time
}

type History struct {
	histories []*_HistoryLine
}

func (h *History) At(n int) string {
	return h.histories[n].text
}

func (h *History) textAndStamp(n int) (string, time.Time) {
	entry := h.histories[n]
	return entry.text, entry.stamp
}

func (h *History) Len() int {
	return len(h.histories)
}

func (h *History) Add(text string) {
	h.histories = append(h.histories, &_HistoryLine{
		text:  text,
		stamp: time.Now(),
	})
}
