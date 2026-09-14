package password

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		plain string
		about Attributes
		want  error
	}{
		{"correct horse battery", Attributes{}, nil},
		{"short1", Attributes{}, ErrTooShort},
		{"password", Attributes{}, ErrTooCommon},
		{"9182736450", Attributes{}, ErrAllNumeric},
		{"seeduser1", Attributes{Username: "seeduser"}, ErrTooSimilar},
		{"unrelated phrase", Attributes{Username: "seeduser"}, nil},
	}
	for _, c := range cases {
		if got := Validate(c.plain, c.about); !errors.Is(got, c.want) {
			t.Errorf("Validate(%q, %+v) = %v, want %v", c.plain, c.about, got, c.want)
		}
	}
}

func TestQuickRatio(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"", "", 1},
		{"abc", "abc", 1},
		{"abc", "xyz", 0},
		{"abcd", "abxy", 0.5},
	}
	for _, c := range cases {
		if got := quickRatio(c.a, c.b); got != c.want {
			t.Errorf("quickRatio(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
