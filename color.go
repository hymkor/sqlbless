package main

import (
	"github.com/nyaosorg/go-readline-ny"
)

type Coloring struct {
	bits int
}

var (
	red   = readline.SGR3(1, 49, 31)
	cyan  = readline.SGR3(1, 49, 36)
	reset = readline.SGR3(22, 49, 39)
)

func (c *Coloring) Init() readline.ColorSequence {
	c.bits = 0
	return reset
}

func (c *Coloring) Next(r rune) readline.ColorSequence {
	const (
		_QUOTED = 1
	)
	newbits := c.bits
	if r == '\'' {
		newbits ^= _QUOTED
	}
	defer func() {
		c.bits = newbits
	}()
	if (c.bits&_QUOTED) != 0 || (newbits&_QUOTED) != 0 {
		return red
	}
	return cyan
}
