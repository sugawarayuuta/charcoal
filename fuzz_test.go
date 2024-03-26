package charcoal_test

import (
	"testing"
	"unicode/utf8"

	. "github.com/sugawarayuuta/charcoal"
)

func Fuzz(f *testing.F) {
	if testing.Short() {
		f.SkipNow()
	}
	for _, ent := range tests {
		if !ent.out {
			continue
		}
		f.Add(ent.in)
	}
	f.Fuzz(func(t *testing.T, in string) {
		if utf8.ValidString(in) != ValidString(in) {
			t.Errorf("utf8.ValidString(%q) != ValidString(%q)", in, in)
		}
		if utf8.Valid([]byte(in)) != Valid([]byte(in)) {
			t.Errorf("utf8.Valid(%q) != Valid(%q)", in, in)
		}
	})
}
