package sqlbless

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hymkor/sqlbless/internal/misc"
)

type Script struct {
	br   *bufio.Reader
	echo io.Writer
	term string
}

func (script *Script) GetKey() (string, error) {
	return "", io.EOF
}

func (script *Script) AutoPilotForCsvi() (getKeyAndSize, bool) {
	return nil, false
}

func (script *Script) Read(context.Context) ([]string, error) {
	var buffer strings.Builder
	quoted := 0
	for {
		ch, _, err := script.br.ReadRune()
		if err != nil {
			code := buffer.String()
			fmt.Fprintln(script.echo, code)
			return []string{code}, err
		}
		if ch == '\r' {
			continue
		} else if ch == '\'' {
			quoted ^= 1
		} else if ch == '"' {
			quoted ^= 2
		}
		buffer.WriteRune(ch)

		if quoted == 0 {
			code := buffer.String()
			term := script.term
			if _, ok := misc.HasTerm(code, term); ok {
				println(code)
				fmt.Fprintln(script.echo, code)
				return []string{code}, nil
			}
		}
	}
}

func (ss *Session) StartFromStdin(ctx context.Context) error {
	script := &Script{
		br:   bufio.NewReader(os.Stdin),
		echo: ss.stdErr,
		term: ss.Term,
	}
	return ss.Loop(ctx, script, true)
}

func (ss *Session) Start(ctx context.Context, fname string) error {
	if fname == "-" {
		return ss.StartFromStdin(ctx)
	}
	fd, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer fd.Close()
	script := &Script{
		br:   bufio.NewReader(fd),
		echo: ss.stdErr,
		term: ss.Term,
	}
	return ss.Loop(ctx, script, true)
}
