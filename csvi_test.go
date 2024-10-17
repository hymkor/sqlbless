package sqlbless

import (
	"testing"
)

func TestToOneLine(t *testing.T) {
	list := []struct {
		source string
		expect string
		prefix string
		suffix string
	}{
		{source: "  select * from t\n\t  where a == '  '",
			expect: "[select * from t where a == '  ']",
			prefix: "[",
			suffix: "]",
		},
	}

	for _, p := range list {
		result := toOneLine(p.source, p.prefix, p.suffix)
		if result != p.expect {
			t.Fatalf("source=%s expect=%s prefix=%s suffix=%s, but %s",
				p.source, p.expect, p.prefix, p.suffix, result)
		}
	}
}
