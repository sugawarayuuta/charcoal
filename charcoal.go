package charcoal

// Valid reports whether buf consists entirely of valid UTF-8-encoded runes.
func Valid(buf []byte) bool {
	s64 := state64{xe0: m80, xed: m80, xf0: m80}
	var idx uint
	for idx+8 <= uint(len(buf)) {
		data := leBytes(buf[idx : idx+8 : idx+8])
		if data&m80 != 0 {
			break
		}
		idx += 8
	}
	for idx+8 <= uint(len(buf)) {
		data := leBytes(buf[idx : idx+8 : idx+8])
		if !s64.add(data) {
			return false
		}
		idx += 8
	}
	var data uint64
	if len(buf) >= 8 {
		shft := 64 - uint(len(buf))%8*8
		data = leBytes(buf[len(buf)-8:len(buf):len(buf)]) >> shft
	} else {
		var shft uint
		for idx < uint(len(buf)) {
			data |= uint64(buf[idx]) << shft
			shft += 8
			idx++
		}
	}
	return (data&m80 == 0 || s64.add(data)) && s64.top&m80 == 0
}

// ValidString reports whether buf consists entirely of valid UTF-8-encoded runes.
func ValidString(buf string) bool {
	s64 := state64{xe0: m80, xed: m80, xf0: m80}
	var idx uint
	for idx+8 <= uint(len(buf)) {
		data := leString(buf[idx : idx+8])
		if data&m80 != 0 {
			break
		}
		idx += 8
	}
	for idx+8 <= uint(len(buf)) {
		data := leString(buf[idx : idx+8])
		if !s64.add(data) {
			return false
		}
		idx += 8
	}
	var data uint64
	if len(buf) >= 8 {
		shft := 64 - uint(len(buf))%8*8
		data = leString(buf[len(buf)-8:]) >> shft
	} else {
		var shft uint
		for idx < uint(len(buf)) {
			data |= uint64(buf[idx]) << shft
			shft += 8
			idx++
		}
	}
	return (data&m80 == 0 || s64.add(data)) && s64.top&m80 == 0
}

func leBytes(buf []byte) uint64 {
	_ = buf[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(buf[0]) | uint64(buf[1])<<8 |
		uint64(buf[2])<<16 | uint64(buf[3])<<24 |
		uint64(buf[4])<<32 | uint64(buf[5])<<40 |
		uint64(buf[6])<<48 | uint64(buf[7])<<56
}

func leString(buf string) uint64 {
	_ = buf[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(buf[0]) | uint64(buf[1])<<8 |
		uint64(buf[2])<<16 | uint64(buf[3])<<24 |
		uint64(buf[4])<<32 | uint64(buf[5])<<40 |
		uint64(buf[6])<<48 | uint64(buf[7])<<56
}
