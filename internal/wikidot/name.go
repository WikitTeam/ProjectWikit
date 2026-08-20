package wikidot

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	DefaultCategory = "_default"
	maxNameLength   = 128
)

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ж': "z",
	'з': "z", 'и': "i", 'й': "i", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "h", 'ц': "c",
	'ч': "c", 'ы': "i", 'э': "e", 'ю': "u", 'я': "a", 'і': "i", 'ї': "i", 'є': "e",
	'ь': "", 'ъ': "",
}

var reserved = []string{"-", "_", "api", "forum", "local--files", "local--code", "local--html", "local--theme"}

func Normalize(fullName string) string {
	var b strings.Builder
	for _, r := range stripAccents(strings.ToLower(fullName)) {
		if sub, ok := translit[r]; ok {
			b.WriteString(sub)
			continue
		}
		b.WriteRune(r)
	}

	s := strings.Trim(dashRuns(b.String()), "-")
	s = strings.Trim(collapseColons(s), ":")

	category, name := Split(s)
	if category == DefaultCategory {
		return name
	}
	return category + ":" + name
}

func Split(fullName string) (category, name string) {
	if c, n, ok := strings.Cut(fullName, ":"); ok {
		return c, n
	}
	return DefaultCategory, fullName
}

func Denormalize(fullName string) string {
	if strings.Contains(fullName, ":") {
		return fullName
	}
	return DefaultCategory + ":" + fullName
}

func NameAllowed(fullName string) bool {
	s := strings.ToLower(fullName)
	if s == "" || slices.Contains(reserved, s) {
		return false
	}
	if utf8.RuneCountInString(s) > maxNameLength {
		return false
	}
	for _, r := range s {
		if !allowed(r) {
			return false
		}
	}
	category, name := Split(s)
	return strings.TrimSpace(category) != "" && strings.TrimSpace(name) != ""
}

func stripAccents(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func dashRuns(s string) string {
	var b strings.Builder
	pending := false
	for _, r := range s {
		if !allowed(r) {
			pending = true
			continue
		}
		if pending {
			b.WriteByte('-')
			pending = false
		}
		b.WriteRune(r)
	}
	if pending {
		b.WriteByte('-')
	}
	return b.String()
}

func collapseColons(s string) string {
	var b strings.Builder
	prev := false
	for _, r := range s {
		if r == ':' && prev {
			continue
		}
		prev = r == ':'
		b.WriteRune(r)
	}
	return b.String()
}

func allowed(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return r == '-' || r == '_' || r == ':'
}
