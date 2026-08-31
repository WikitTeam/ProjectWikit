package articlepage

import "testing"

func TestUnwrapParagraphs(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<p>one line</p>", "one line"},
		{"<p>a</p>\n<span>b</span>\n<p>c</p>", "a\n<span>b</span>\nc"},
		{"", ""},
		{"no markup", "no markup"},
	}
	for _, c := range cases {
		if got := unwrapParagraphs(c.in); got != c.want {
			t.Errorf("unwrapParagraphs(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
