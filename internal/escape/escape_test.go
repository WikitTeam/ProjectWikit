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

func TestURLQuoteKeepsPythonSafeSet(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abcXYZ019", "abcXYZ019"},
		{"_.-~/", "_.-~/"},
		{" ", "%20"},
		{":", "%3A"},
		{"&+,;=@$", "%26%2B%2C%3B%3D%40%24"},
		{"中", "%E4%B8%AD"},
	}
	for _, c := range cases {
		if got := URLQuote(c.in); got != c.want {
			t.Errorf("URLQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
