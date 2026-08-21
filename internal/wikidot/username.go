package wikidot

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// CanonicalizeUsername mirrors canonicalize_username in web/models/users.py.
// Unlike Normalize it keeps non-Latin letters, so a Chinese name canonicalizes
// to itself rather than to an empty string (FINDINGS §12).
//
// Python's [\W_]+ is every run that is neither a letter nor a number, because
// \W excludes the underscore that the class then adds back.
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
