package history

import (
	"time"
)

type Line struct {
	text  string
	stamp time.Time
}

type History struct {
	histories []*Line
}

func (h *History) At(n int) string {
	return h.histories[n].text
}

func (h *History) TextAndStamp(n int) (string, time.Time) {
	entry := h.histories[n]
	return entry.text, entry.stamp
}

func (h *History) Len() int {
	return len(h.histories)
}

func (h *History) Add(text string) {
	h.histories = append(h.histories, &Line{
		text:  text,
		stamp: time.Now(),
	})
}
