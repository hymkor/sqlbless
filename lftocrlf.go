package main

import (
	"io"

	"golang.org/x/text/transform"
)

type lfToCrlf struct{}

func (t lfToCrlf) Reset() {}

func (f lfToCrlf) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for _, c := range src {
		if c == '\n' {
			if len(dst) < 2 {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[0] = '\r'
			dst[1] = '\n'
			dst = dst[2:]
			nDst += 2
		} else {
			if len(dst) < 1 {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[0] = c
			dst = dst[1:]
			nDst++
		}
		nSrc++
	}
	return nDst, nSrc, nil
}

type FilterSource interface {
	Write([]byte) (int, error)
	Name() string
	Close() error
}

type Filter struct {
	body   FilterSource
	filter io.WriteCloser
}

func (s *Filter) Write(b []byte) (int, error) {
	if s.filter != nil {
		return s.filter.Write(b)
	} else {
		return s.body.Write(b)

	}
}

func (s *Filter) Close() error {
	if s.filter != nil {
		s.filter.Close()
	}
	return s.body.Close()
}

func (s *Filter) Name() string {
	return s.body.Name()
}

func newFilter(fd FilterSource) *Filter {
	filter := transform.NewWriter(fd, lfToCrlf{})
	return &Filter{
		filter: filter,
		body:   fd,
	}
}
