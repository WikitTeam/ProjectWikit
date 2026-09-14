package password

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

//go:embed common-passwords.txt.gz
var commonList []byte

const MinLength = 8

const maxSimilarity = 0.7

var (
	ErrTooShort   = errors.New("password: shorter than the minimum")
	ErrTooCommon  = errors.New("password: on the common list")
	ErrAllNumeric = errors.New("password: entirely numeric")
	ErrTooSimilar = errors.New("password: too close to the account's own details")
)

type Attributes struct {
	Username    string
	DisplayName string
	FirstName   string
	LastName    string
	Email       string
}

func (a Attributes) values() []string {
	return []string{a.Username, a.DisplayName, a.FirstName, a.LastName, a.Email}
}

func Validate(plain string, about Attributes) error {
	if err := tooSimilar(plain, about); err != nil {
		return err
	}
	if utf8.RuneCountInString(plain) < MinLength {
		return ErrTooShort
	}
	if common(plain) {
		return ErrTooCommon
	}
	if allNumeric(plain) {
		return ErrAllNumeric
	}
	return nil
}

var nonWord = regexp.MustCompile(`\W+`)

func tooSimilar(plain string, about Attributes) error {
	lower := strings.ToLower(plain)
	for _, value := range about.values() {
		if value == "" {
			continue
		}
		for _, part := range append(nonWord.Split(value, -1), value) {
			if part == "" {
				continue
			}
			if quickRatio(lower, strings.ToLower(part)) >= maxSimilarity {
				return ErrTooSimilar
			}
		}
	}
	return nil
}

func quickRatio(a, b string) float64 {
	if len(a)+len(b) == 0 {
		return 1
	}
	counts := map[rune]int{}
	for _, r := range b {
		counts[r]++
	}
	matches := 0
	for _, r := range a {
		if counts[r] > 0 {
			counts[r]--
			matches++
		}
	}
	return 2 * float64(matches) / float64(utf8.RuneCountInString(a)+utf8.RuneCountInString(b))
}

func allNumeric(plain string) bool {
	if plain == "" {
		return false
	}
	for _, r := range plain {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

var loadCommon = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	reader, err := gzip.NewReader(bytes.NewReader(commonList))
	if err != nil {
		return out
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		out[strings.TrimSpace(scanner.Text())] = true
	}
	return out
})

func common(plain string) bool {
	return loadCommon()[strings.ToLower(strings.TrimSpace(plain))]
}
