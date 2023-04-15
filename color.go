package main

import (
	"github.com/nyaosorg/go-readline-ny"
)

type Coloring struct {
	bits int
}

func (c *Coloring) Init() readline.ColorSequence {
	c.bits = 0
	return readline.DefaultForeGroundColor
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
