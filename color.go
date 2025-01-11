package sqlbless

import (
	"time"
)

type _HistoryLine struct {
	text  string
	stamp time.Time
}

type History struct {
	histories []*_HistoryLine
}

func (h *History) At(n int) string {
	return h.histories[n].text
}

func (h *History) textAndStamp(n int) (string, time.Time) {
	entry := h.histories[n]
	return entry.text, entry.stamp
}

func (h *History) Len() int {
	return len(h.histories)
}

func (h *History) Add(text string) {
	h.histories = append(h.histories, &_HistoryLine{
		text:  text,
		stamp: time.Now(),
	})
}
