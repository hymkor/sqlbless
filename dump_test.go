package main

import (
	"io"
	"strings"
	"testing"

	"golang.org/x/text/transform"
)

func TestLfToCrLf(t *testing.T) {
	source := "hogehoge\nahahah\nihihi"
	expect := "hogehoge\r\nahahah\r\nihihi"

	var resultBuffer strings.Builder
	w := transform.NewWriter(&resultBuffer, lfToCrlf{})
	io.Copy(w, strings.NewReader(source))
	result := resultBuffer.String()

	if result != expect {
		t.Fatalf("expect '%v', but '%v'", expect, result)
	}
}
