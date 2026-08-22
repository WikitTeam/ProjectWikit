// Package escape reproduces Django's HTML escaping.
package escape

import "strings"

// Django writes ' as &#x27; where html.EscapeString writes &#39;, and the
// acceptance test for the whole renderer is byte-identical output.
var replacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
)

func HTML(s string) string { return replacer.Replace(s) }
