package misc

import (
	"fmt"
	"io"
	"strings"
	"time"
)

var undq = strings.NewReplacer(
	`""`, `"`,
	`"`, ``,
)

func CutField(s string) (string, string) {
	s = strings.TrimLeft(s, " \n\r\t\v")
	i := 0
	q := false
	for len(s) > i && (q || (s[i] != ' ' && s[i] != '\n' && s[i] != '\r' && s[i] != '\t' && s[i] != '\v')) {
		if s[i] == '"' {
			q = !q
		}
		i++
	}
	return undq.Replace(s[:i]), s[i:]
}

func Echo(spool io.Writer, query string) {
	EchoPrefix(spool, "", query)
}

func EchoPrefix(spool io.Writer, prefix, query string) {
	if spool == nil {
		return
	}
	fmt.Fprintf(spool, "### <%s> ###\n", time.Now().Local().Format(time.DateTime))
	query = strings.TrimRight(query, "\n")
	for {
		var line string
		var next bool
		line, query, next = strings.Cut(query, "\n")
		fmt.Fprintf(spool, "# %s%s\n", prefix, line)
		if !next {
			break
		}
	}
}

// hasTerm is similar with strings.HasSuffix, but ignores cases when comparing and returns the trimed string and the boolean indicating trimed or not
func HasTerm(s, term string) (string, bool) {
	s = strings.TrimRight(s, " \r\n\t\v")
	from := len(s) - len(term)
	if 0 <= from && from < len(s) && strings.EqualFold(s[from:], term) {
		return s[:from], true
	}
	return s, false
}
