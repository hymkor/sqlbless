package sqlbless

import (
	"io"

	"golang.org/x/text/transform"
)

type lfToCrlfTransformer struct{}

func (t lfToCrlfTransformer) Reset() {}

func (f lfToCrlfTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
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

type writeNameCloser interface {
	Write([]byte) (int, error)
	Name() string
	Close() error
}

type lfToCrlf struct {
	body   writeNameCloser
	filter io.WriteCloser
}

func (s *lfToCrlf) Write(b []byte) (int, error) {
	if s.filter != nil {
		return s.filter.Write(b)
	} else {
		return s.body.Write(b)

	}
}

func (s *lfToCrlf) Close() error {
	if s.filter != nil {
		s.filter.Close()
	}
	return s.body.Close()
}

func (s *lfToCrlf) Name() string {
	return s.body.Name()
}

func newLfToCrlf(fd writeNameCloser) *lfToCrlf {
	filter := transform.NewWriter(fd, lfToCrlfTransformer{})
	return &lfToCrlf{
		filter: filter,
		body:   fd,
	}
}
