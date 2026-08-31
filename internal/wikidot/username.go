package wikidot

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Unlike Normalize this keeps non-Latin letters, so a Chinese name
// canonicalizes to itself rather than to an empty string. Every run that is
// neither a letter nor a number collapses, the underscore included.
func CanonicalizeUsername(name string) string {
	var b strings.Builder
	pending := false
	for _, r := range strings.ToLower(norm.NFKC.String(name)) {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			pending = true
			continue
		}
		if pending && b.Len() > 0 {
			b.WriteRune('-')
		}
		pending = false
		b.WriteRune(r)
	}
	return b.String()
}
