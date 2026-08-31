// Package wikinum parses the numbers a wikitext author typed, which tolerate
// surrounding space, a sign and underscores between digits.
package wikinum

import (
	"errors"
	"strconv"
	"strings"
)

var ErrNotANumber = errors.New("wikinum: not a number")

func Int(s string) (int, error) {
	s = strings.TrimSpace(s)
	body := strings.TrimPrefix(strings.TrimPrefix(s, "+"), "-")
	if body == "" || strings.HasPrefix(body, "_") || strings.HasSuffix(body, "_") {
		return 0, ErrNotANumber
	}
	if strings.Contains(body, "__") {
		return 0, ErrNotANumber
	}
	for _, r := range body {
		if r != '_' && (r < '0' || r > '9') {
			return 0, ErrNotANumber
		}
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, "_", ""))
	if err != nil {
		return 0, ErrNotANumber
	}
	return n, nil
}

// Go would take a hexadecimal float that this must not, so the digits are
// checked first.
func Float(s string) (float64, error) {
	if n, err := Int(s); err == nil {
		return float64(n), nil
	}
	s = strings.TrimSpace(s)
	lowered := strings.ToLower(s)
	if strings.Contains(lowered, "x") || strings.Contains(lowered, "p") {
		return 0, ErrNotANumber
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, "_", ""), 64)
	if err != nil {
		return 0, ErrNotANumber
	}
	return f, nil
}
