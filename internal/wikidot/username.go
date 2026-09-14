package wikidot

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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

const (
	displayNameMax = 50

	fallbackPrefix = "wkt-uid"
)

var reservedUsername = regexp.MustCompile(`^` + fallbackPrefix + `-\d+(-\d+)*$`)

func ReservedUsername(name string) bool { return reservedUsername.MatchString(name) }

func FallbackUsername(userID int64, suffix int) string {
	base := fallbackPrefix + "-" + strconv.FormatInt(userID, 10)
	if suffix < 2 {
		return base
	}
	return base + "-" + strconv.Itoa(suffix)
}

var whitespace = regexp.MustCompile(`\s+`)

func NormalizeDisplayName(name string) string {
	return strings.TrimSpace(whitespace.ReplaceAllString(norm.NFKC.String(name), " "))
}

type DisplayNameProblem int

const (
	DisplayNameOK DisplayNameProblem = iota
	DisplayNameEmpty
	DisplayNameTooLong
	DisplayNameInvisible
	DisplayNameOddSpace
	DisplayNameLeadingMark
)

func ValidateDisplayName(name string) DisplayNameProblem {
	if name == "" {
		return DisplayNameEmpty
	}
	if utf8.RuneCountInString(name) > displayNameMax {
		return DisplayNameTooLong
	}
	for _, r := range name {
		switch {
		case unicode.In(r, unicode.Cc, unicode.Cf, unicode.Cs, unicode.Co, unicode.Zl, unicode.Zp):
			return DisplayNameInvisible
		case unassigned(r):
			return DisplayNameInvisible
		case unicode.In(r, unicode.Zs) && r != ' ':
			return DisplayNameOddSpace
		}
	}
	first, _ := utf8.DecodeRuneInString(name)
	if unicode.In(first, unicode.Mn, unicode.Mc) {
		return DisplayNameLeadingMark
	}
	return DisplayNameOK
}

func unassigned(r rune) bool {
	for _, table := range unicode.Categories {
		if unicode.Is(table, r) {
			return false
		}
	}
	return true
}
