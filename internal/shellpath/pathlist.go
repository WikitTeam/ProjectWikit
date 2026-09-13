package shellpath

import (
	"path/filepath"
	"strings"
)

// These edit the text of a Windows Path value. They live apart from the
// registry so the rules can be tested anywhere.

func splitPath(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ";") {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}

func sameEntry(a, b string) bool {
	clean := func(s string) string {
		return strings.ToLower(strings.TrimRight(filepath.Clean(strings.TrimSpace(s)), `\/`))
	}
	return clean(a) == clean(b)
}

func pathHas(value, dir string) bool {
	for _, part := range splitPath(value) {
		if sameEntry(part, dir) {
			return true
		}
	}
	return false
}

func pathWithout(value, dir string) string {
	var kept []string
	for _, part := range splitPath(value) {
		if !sameEntry(part, dir) {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ";")
}

func pathWith(value, dir string) string {
	if pathHas(value, dir) {
		return value
	}
	parts := append(splitPath(value), dir)
	return strings.Join(parts, ";")
}
