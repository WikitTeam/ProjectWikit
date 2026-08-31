package escape

import (
	"fmt"
	"strings"
)

const (
	jsSpecials   = "\\'\"><&=-;`"
	lineSep      = '\u2028'
	paragraphSep = '\u2029'
)

// Go's template.JSEscapeString covers a different set of characters and spells
// them differently, so it cannot stand in here.
func JS(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == lineSep || r == paragraphSep || strings.ContainsRune(jsSpecials, r) {
			fmt.Fprintf(&b, "\\u%04X", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
