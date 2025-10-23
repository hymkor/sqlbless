package sqlbless

import (
	"regexp"
	"strings"

	"github.com/mattn/go-tty"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/auto"
	"github.com/nyaosorg/go-readline-ny/keys"

	"github.com/hymkor/csvi"
	"github.com/hymkor/go-multiline-ny"
	"github.com/hymkor/go-multiline-ny/completion"

	"github.com/hymkor/sqlbless/internal/misc"
	"github.com/hymkor/sqlbless/internal/sqlcompletion"
)

type reserveWordPattern map[string]struct{}

var rxWords = regexp.MustCompile(`\b\w+\b`)

func (h reserveWordPattern) FindAllStringIndex(s string, n int) [][]int {
	matches := rxWords.FindAllStringIndex(s, n)
	for i := len(matches) - 1; i >= 0; i-- {
		word := s[matches[i][0]:matches[i][1]]
		if _, ok := h[strings.ToUpper(word)]; !ok {
			copy(matches[i:], matches[i+1:])
			matches = matches[:len(matches)-1]
		}
	}
	return matches
}

func newReservedWordPattern(list ...string) reserveWordPattern {
	m := reserveWordPattern{}
	for _, word := range list {
		m[strings.ToUpper(word)] = struct{}{}
	}
	return m
}

var o = struct{}{}

var oneLineCommands = map[string]struct{}{
	`DESC`:    o,
	`EDIT`:    o,
	`EXIT`:    o,
	`HISTORY`: o,
	`HOST`:    o,
	`QUIT`:    o,
	`REM`:     o,
	`SPOOL`:   o,
	`START`:   o,
	`\D`:      o,
}

func isOneLineCommand(cmdLine string) bool {
	first, _ := misc.CutField(cmdLine)
	first = strings.ToUpper(first)
	first = strings.TrimRight(first, ";")
	_, ok := oneLineCommands[first]
	return ok
}

type interactiveIn struct {
	*multiline.Editor
	tty       misc.GetKeyAndSize
	csviPilot csvi.Pilot
}

func (*interactiveIn) CanCloseInTransaction() bool { return false }

func (i *interactiveIn) GetKey() (string, error) {
	if i.tty != nil {
		return i.tty.GetKey()
	}
	tt, err := tty.Open()
	if err != nil {
		return "", err
	}
	defer tt.Close()
	return readline.GetKey(tt)
}

func (i *interactiveIn) AutoPilotForCsvi() (csvi.Pilot, bool) {
	return i.csviPilot, (i.csviPilot != nil)
}

func (ss *session) newInteractiveIn() *interactiveIn {
	var editor multiline.Editor

	editor.ResetColor = "\x1B[0m"
	editor.DefaultColor = "\x1B[39;49;1m"
	editor.Highlight = []readline.Highlight{
		{Pattern: newReservedWordPattern("ALTER", "COMMIT", "CREATE", "DELETE", "DESC", "DROP", "EXIT", "HISTORY", "INSERT", "QUIT", "REM", "ROLLBACK", "SELECT", "SPOOL", "START", "TRUNCATE", "UPDATE", "AND", "FROM", "INTO", "OR", "WHERE"), Sequence: "\x1B[36;49;1m"},
		{Pattern: regexp.MustCompile(`[0-9]+`), Sequence: "\x1B[35;49;1m"},
		{Pattern: regexp.MustCompile(`/\*.*?\*/`), Sequence: "\x1B[33;49;22m"},
		{Pattern: regexp.MustCompile(`"[^"]*"|"[^"]*$`), Sequence: "\x1B[31;49;1m"},
		{Pattern: regexp.MustCompile(`'[^']*'|'[^']*$`), Sequence: "\x1B[35;49;1m"},
	}

	if ss.SubmitByEnter {
		editor.SwapEnter()
	}
	var tty misc.GetKeyAndSize
	var csviPilot csvi.Pilot
	if ss.Auto != "" {
		text := strings.ReplaceAll(ss.Auto, "||", "\n") // "||" -> Ctrl-J(Commit)
		text = strings.ReplaceAll(text, "|", "\r")      // "|" -> Ctrl-M (NewLine)
		if text[len(text)-1] != '\n' {                  // EOF -> Ctrl-J(Commit)
			text = text + "\n"
		}
		tty1 := &auto.Pilot{
			Text: strings.Split(text, ""),
		}
		editor.LineEditor.Tty = tty1
		tty = tty1
		csviPilot = misc.AutoCsvi{tty1}
	}
	editor.SetPredictColor(readline.PredictColorBlueItalic)
	editor.SetHistory(ss.history)
	editor.SetWriter(ss.termOut)

	editor.BindKey(keys.CtrlI, &completion.CmdCompletionOrList{
		Enclosure:  `"'`,
		Delimiter:  ",",
		Postfix:    " ",
		Candidates: sqlcompletion.New(ss.Dialect, ss.conn),
	})
	editor.SubmitOnEnterWhen(func(lines []string, csrline int) bool {
		if len(lines) > 0 && isOneLineCommand(lines[0]) {
			return true
		}
		for {
			last := strings.TrimRight(lines[len(lines)-1], " \r\n\t\v")
			if last != "" || len(lines) <= 1 {
				if len(ss.Term) == 1 {
					_, ok := misc.HasTerm(last, ss.Term)
					return ok
				} else {
					return strings.EqualFold(last, ss.Term)
				}
			}
			lines = lines[:len(lines)-1]
		}
	})
	return &interactiveIn{
		Editor:    &editor,
		tty:       tty,
		csviPilot: csviPilot,
	}
}
