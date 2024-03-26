package charcoal_test

import (
	"bytes"
	"os"
	"strconv"
	"testing"
	"unicode/utf8"

	. "github.com/sugawarayuuta/charcoal"
)

type (
	test struct {
		in  string
		out bool
	}
)

var (
	tests = []test{
		{"", true},
		{"a", true},
		{"abc", true},
		{"Ж", true},
		{"ЖЖ", true},
		{"брэд-ЛГТМ", true},
		{"☺☻☹", true},
		{"aa\xe2", false},
		{string([]byte{66, 250}), false},
		{string([]byte{66, 250, 67}), false},
		{"a\uFFFDb", true},
		{string("\xF4\x8F\xBF\xBF"), true},      // U+10FFFF
		{string("\xF4\x90\x80\x80"), false},     // U+10FFFF+1; out of range
		{string("\xF7\xBF\xBF\xBF"), false},     // 0x1FFFFF; out of range
		{string("\xFB\xBF\xBF\xBF\xBF"), false}, // 0x3FFFFFF; out of range
		{string("\xc0\x80"), false},             // U+0000 encoded in two bytes: incorrect
		{string("\xed\xa0\x80"), false},         // U+D800 high surrogate (sic)
		{string("\xed\xbf\xbf"), false},         // U+DFFF low surrogate (sic)
	}
)

func TestValid(t *testing.T) {
	for _, ent := range tests {
		if Valid([]byte(ent.in)) != ent.out {
			t.Errorf("Valid(%q) = %t; want %t", ent.in, !ent.out, ent.out)
		}
		if ValidString(ent.in) != ent.out {
			t.Errorf("ValidString(%q) = %t; want %t", ent.in, !ent.out, ent.out)
		}
	}
}

func BenchmarkValid(b *testing.B) {
	data, err := os.ReadFile("./testdata/unicode.json")
	if err != nil {
		b.Fatal(err)
	}
	benchmarkValid(b, "ascii-small", []byte("0123456789"))
	benchmarkValid(b, "ascii-large", bytes.Repeat([]byte("0123456789"), 10000))
	benchmarkValid(b, "kanji-small", []byte("日本語日本語日本語日"))
	benchmarkValid(b, "kanji-large", bytes.Repeat([]byte("日本語日本語日本語日"), 3333))
	benchmarkValid(b, "unicode.json", data)
}

func benchmarkValid(b *testing.B, name string, buf []byte) {
	for i := 0; i <= 7; i++ {
		name2 := name + "+" + strconv.Itoa(i)

		// Make a copy of buf, but shifted by i bytes to check with buffer start not aligned on int64
		buf2 := make([]byte, i+len(buf))
		buf2 = buf2[i:]
		copy(buf2, buf)
		if !bytes.Equal(buf2, buf) {
			b.Fatal("bug!")
		}

		b.Run("standard:"+name2, func(b *testing.B) {
			b.SetBytes(int64(len(buf2)))
			for try := 0; try < b.N; try++ {
				ok := utf8.Valid(buf2)
				if !ok {
					b.Fatal("!ok")
				}
			}
		})
		b.Run("charcoal:"+name2, func(b *testing.B) {
			b.SetBytes(int64(len(buf2)))
			for try := 0; try < b.N; try++ {
				ok := Valid(buf2)
				if !ok {
					b.Fatal("!ok")
				}
			}
		})
	}
}
