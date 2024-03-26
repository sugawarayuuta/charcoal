package charcoal

import (
	"math/bits"
)

type (
	// state64 holds the current validation state.
	state64 struct {
		// Records effects of special starting bytes.
		xe0, xed, xf0, xf4 uint64
		// Effects of special starting bytes but combined.
		all uint64
		// Sequences that exist across multiple words.
		top uint64
	}
)

const (
	// Sorted list of constant masks.
	// m01 looks like 0x0101... in hexadecimal.
	// Multiplying bytes to it duplicates it to
	// every byte in the word.
	m01 = ^uint64(0) / 255
	m0b = m01 * 0x0b
	m0c = m01 * 0x0c
	m10 = m01 * 0x10
	m41 = m01 * 0x41
	m60 = m01 * 0x60
	m6d = m01 * 0x6d
	m70 = m01 * 0x70
	m7f = m01 * 0x7f
	m80 = m01 * 0x80
	mf0 = m01 * 0xf0
)

func (s64 *state64) add(inp uint64) bool {
	low := inp & m7f
	pop := m80 - inp>>7&m01
	// 1) Detect 0xc0 and 0xc1. It is always
	// an error if present.
	err := ^(low | m01 ^ m41 + pop)
	// 2) Detect conditional errors (starts with
	// 0xe0, 0xed, 0xf0, 0xf4).
	xf0 := pop + (low | m10 ^ m70)
	xed := pop + (low ^ m6d)
	xf4 := inp & (low + m0c)
	all := xf4 | ^(xed & xf0)
	// 3) Do the relatively expensice computation
	// only if needed.
	if (all|s64.all)&m80 != 0 {
		xe0 := pop + (low ^ m60)
		xf5 := inp & (low + m0b)
		xf0 &= xe0
		xf4 ^= xf5
		s64.xe0, xe0 = xe0, xe0<<8|s64.xe0>>56
		s64.xed, xed = xed, xed<<8|s64.xed>>56
		s64.xf0, xf0 = xf0, xf0<<8|s64.xf0>>56
		s64.xf4, xf4 = xf4, xf4<<8|s64.xf4>>56
		x60, x70 := low+m60, low+m70
		not := (xe0 | x60) & (xed | ^x60) & (xf0 | x70)
		err |= xf5 | ^not | xf4&x70
	}
	s64.all = all
	// 4) Detect ill-formed sequences by their
	// orders and prefixes.
	one := mf0 &^ inp
	one |= one >> 1
	one |= one >> 2
	top, btm := bits.Mul64(m70&^one, 0x08040200)
	err |= s64.top | btm ^ inp&(one<<1)
	s64.top = top
	return err&m80 == 0
}
