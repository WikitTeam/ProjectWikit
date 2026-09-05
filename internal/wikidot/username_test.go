package wikidot

import (
	"strings"
	"testing"
)

// Vectors generated from canonicalize_username in web/models/users.py.
func TestCanonicalizeUsername(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Admin", "admin"},
		{"ADMIN", "admin"},
		{"John Doe", "john-doe"},
		{"  spaced  ", "spaced"},
		{"a__b", "a-b"},
		{"!!!", ""},
		{"", ""},
		{"测试用户", "测试用户"},
		{"测试 用户", "测试-用户"},
		{"Ｆｕｌｌｗｉｄｔｈ", "fullwidth"},
		{"user-name", "user-name"},
		{"--lead--trail--", "lead-trail"},
		{"Ünïcödé", "ünïcödé"},
		{"ｱｲｳ", "アイウ"},
		{"a.b.c", "a-b-c"},
		{"12345", "12345"},
		{"café", "café"},
		{"Иван Петров", "иван-петров"},
		{"a-_-b", "a-b"},
		{"x²", "x2"},
		{"ﬁle", "file"},
		{"ᴬ", "a"},
	}
	for _, c := range cases {
		if got := CanonicalizeUsername(c.in); got != c.want {
			t.Errorf("CanonicalizeUsername(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Ada  Lovelace ", "Ada Lovelace"},
		{"Ａｄａ", "Ada"},
		{"a b", "a b"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizeDisplayName(c.in); got != c.want {
			t.Errorf("NormalizeDisplayName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidateDisplayName(t *testing.T) {
	cases := []struct {
		in   string
		want DisplayNameProblem
	}{
		{"Ada", DisplayNameOK},
		{"", DisplayNameEmpty},
		{strings.Repeat("a", 51), DisplayNameTooLong},
		{"a​b", DisplayNameInvisible},
		{"a b", DisplayNameOddSpace},
		{"́ada", DisplayNameLeadingMark},
	}
	for _, c := range cases {
		if got := ValidateDisplayName(c.in); got != c.want {
			t.Errorf("ValidateDisplayName(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestReservedUsername(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"wkt-uid-12", true},
		{"wkt-uid-12-2", true},
		{"wkt-uid", false},
		{"ada", false},
	}
	for _, c := range cases {
		if got := ReservedUsername(c.in); got != c.want {
			t.Errorf("ReservedUsername(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
