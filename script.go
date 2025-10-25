package sqlbless

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hymkor/csvi"

	"github.com/hymkor/sqlbless/internal/misc"
)

type scriptIn struct {
	br   *bufio.Reader
	echo io.Writer
	term string
}

func (*scriptIn) CanCloseInTransaction() bool                 { return true }
func (*scriptIn) ShouldRecordHistory() bool                   { return false }
func (*scriptIn) SetPrompt(func(io.Writer, int) (int, error)) {}
func (*scriptIn) OnErrorAbort() bool                          { return true }

func (script *scriptIn) GetKey() (string, error) {
	return "", io.EOF
}

func (script *scriptIn) AutoPilotForCsvi() (csvi.Pilot, bool) {
	return nil, false
}

func (script *scriptIn) Read(context.Context) ([]string, error) {
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

func (ss *session) StartFromStdin(ctx context.Context) error {
	script := &scriptIn{
		br:   bufio.NewReader(os.Stdin),
		echo: ss.stdErr,
		term: ss.Term,
	}
	return ss.Loop(ctx, script)
}

func (ss *session) Start(ctx context.Context, fname string) error {
	if fname == "-" {
		return ss.StartFromStdin(ctx)
	}
	fd, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer fd.Close()
	script := &scriptIn{
		br:   bufio.NewReader(fd),
		echo: ss.stdErr,
		term: ss.Term,
	}
	return ss.Loop(ctx, script)
}
