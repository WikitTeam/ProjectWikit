package wikidot

import "testing"

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
