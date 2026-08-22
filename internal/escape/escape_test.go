package escape

import "testing"

func TestHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"plain", "plain"},
		{"<b>", "&lt;b&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&#x27;s"},
		{"a&b", "a&amp;b"},
		{"&lt;", "&amp;lt;"},
		{"中文 <b>", "中文 &lt;b&gt;"},
	}
	for _, c := range cases {
		if got := HTML(c.in); got != c.want {
			t.Errorf("HTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
