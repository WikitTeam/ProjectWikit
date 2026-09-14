package escape

import "testing"

func unicodeEscape(hex string) string { return `\` + "u" + hex }

func TestJS(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"plain", "plain"},
		{`\`, unicodeEscape("005C")},
		{"'", unicodeEscape("0027")},
		{`"`, unicodeEscape("0022")},
		{">", unicodeEscape("003E")},
		{"<", unicodeEscape("003C")},
		{"&", unicodeEscape("0026")},
		{"=", unicodeEscape("003D")},
		{"-", unicodeEscape("002D")},
		{";", unicodeEscape("003B")},
		{"`", unicodeEscape("0060")},
		{"\n", unicodeEscape("000A")},
		{"\x00", unicodeEscape("0000")},
		{string(rune(0x2028)), unicodeEscape("2028")},
		{string(rune(0x2029)), unicodeEscape("2029")},
		{"G-AB123", "G" + unicodeEscape("002D") + "AB123"},
		{"中文", "中文"},
		{"a b", "a b"},
	}
	for _, c := range cases {
		if got := JS(c.in); got != c.want {
			t.Errorf("JS(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
