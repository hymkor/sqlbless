package lftocrlf

import (
	"io"
	"strings"
	"testing"
)

type writer struct {
	w       io.Writer
	crIsCut bool
}

func (this *writer) Write(b []byte) (int, error) {
	var n int
	var err error
	n, this.crIsCut, err = write(this.w, this.crIsCut, b)
	return n, err
}

func TestLfToCrLf(t *testing.T) {
	type tc struct {
		source []string
		expect string
	}

	cases := []tc{
		tc{ // LF to CRLF
			source: []string{"foo\nbar\nbaz"},
			expect: "foo\r\nbar\r\nbaz",
		},
		tc{ // CRLF to CRLF
			source: []string{"foo\r\nbar\r\nbaz"},
			expect: "foo\r\nbar\r\nbaz",
		},
		tc{ // CR""LF to CRLF
			source: []string{"foo\r", "\nbar\nbaz"},
			expect: "foo\r\nbar\r\nbaz",
		},
		tc{ // CR to CR
			source: []string{"foo\r", "bar\nbaz"},
			expect: "foo\rbar\r\nbaz",
		},
	}

	for i, case1 := range cases {
		println("try:", i+1)
		var b strings.Builder
		w := &writer{w: &b}
		for _, src := range case1.source {
			io.Copy(w, strings.NewReader(src))
		}
		result := b.String()

		if result != case1.expect {
			t.Fatalf("(%d) expect '%v', but '%v'",
				i+1,
				[]byte(case1.expect),
				[]byte(result))
		}
	}
}
