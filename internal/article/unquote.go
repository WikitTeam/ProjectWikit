package article

import (
	"strings"
	"unicode/utf8"
)

// An escape unquote cannot read stays in the text, and bytes that are not UTF-8
// turn into U+FFFD.
func unquote(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	buf := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) {
			if b, ok := hexByte(s[i+1], s[i+2]); ok {
				buf = append(buf, b)
				i += 3
				continue
			}
		}
		buf = append(buf, s[i])
		i++
	}
	return replaceInvalid(buf)
}

func hexByte(hi, lo byte) (byte, bool) {
	h, ok := hexDigit(hi)
	if !ok {
		return 0, false
	}
	l, ok := hexDigit(lo)
	if !ok {
		return 0, false
	}
	return h<<4 | l, true
}

func hexDigit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func replaceInvalid(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var out strings.Builder
	for i := 0; i < len(b); {
		if r, size := utf8.DecodeRune(b[i:]); r != utf8.RuneError || size > 1 {
			out.Write(b[i : i+size])
			i += size
			continue
		}
		out.WriteRune(utf8.RuneError)
		i += subpartLen(b[i:])
	}
	return out.String()
}

// One U+FFFD stands for a whole maximal subpart, so a truncated sequence costs
// one replacement rather than one per byte.
func subpartLen(b []byte) int {
	var want int
	var lo, hi byte
	switch c := b[0]; {
	case c >= 0xc2 && c <= 0xdf:
		want, lo, hi = 2, 0x80, 0xbf
	case c == 0xe0:
		want, lo, hi = 3, 0xa0, 0xbf
	case c >= 0xe1 && c <= 0xec, c == 0xee, c == 0xef:
		want, lo, hi = 3, 0x80, 0xbf
	case c == 0xed:
		want, lo, hi = 3, 0x80, 0x9f
	case c == 0xf0:
		want, lo, hi = 4, 0x90, 0xbf
	case c >= 0xf1 && c <= 0xf3:
		want, lo, hi = 4, 0x80, 0xbf
	case c == 0xf4:
		want, lo, hi = 4, 0x80, 0x8f
	default:
		return 1
	}
	if len(b) < 2 || b[1] < lo || b[1] > hi {
		return 1
	}
	n := 2
	for n < want && n < len(b) && b[n] >= 0x80 && b[n] <= 0xbf {
		n++
	}
	return n
}
