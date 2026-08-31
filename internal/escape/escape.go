// Package escape holds the HTML and URL escapes these pages use, which the
// standard library spells differently.
package escape

import "strings"

// These pages spell ' as &#x27; where html.EscapeString writes &#39;, and the
// whole renderer is checked on byte-identical output.
var replacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
)

func HTML(s string) string { return replacer.Replace(s) }

const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.-~/"

// URLQuote is urllib.parse.quote with its default safe="/". Neither
// url.PathEscape nor url.QueryEscape matches it.
func URLQuote(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		const hex = "0123456789ABCDEF"
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0F])
	}
	return b.String()
}
