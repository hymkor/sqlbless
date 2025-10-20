package misc

import (
	"testing"
)

func TestCutField(t *testing.T) {
	tc := [][3]string{
		[...]string{`cut words`, "cut", " words"},
		[...]string{`"cut words" foo`, "cut words", " foo"},
		[...]string{`"cut"" words" foo`, `cut" words`, " foo"},
	}
	for i, tc1 := range tc {
		result1, result2 := CutField(tc1[0])
		if result1 != tc1[1] || result2 != tc1[2] {
			t.Fatalf("%d: %#v: expect %#v and %#v, but %#v and %#v",
				i, tc1[0], tc1[1], tc1[2], result1, result2)
		}
	}
}
