package ipban

import (
	"net/http/httptest"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"203.0.113.7", "203.0.113.7/32", true},
		{" 203.0.113.7 ", "203.0.113.7/32", true},
		{"203.0.113.7/24", "203.0.113.0/24", true},
		{"2001:db8::1", "2001:db8::1/128", true},
		{"2001:db8::1/32", "2001:db8::/32", true},
		{"::ffff:203.0.113.7", "203.0.113.7/32", true},
		{"::ffff:203.0.113.0/120", "203.0.113.0/24", true},
		{"0.0.0.0/0", "", false},
		{"::/0", "", false},
		{"fe80::1%eth0", "", false},
		{"203.0.113", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := Parse(c.in)
		if ok != c.ok {
			t.Errorf("Parse(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && got.String() != c.want {
			t.Errorf("Parse(%q) = %q, want %q", c.in, got.String(), c.want)
		}
	}
}

func TestGuarded(t *testing.T) {
	cases := []struct {
		method, path string
		want         bool
	}{
		{"GET", "/some-page", false},
		{"HEAD", "/-/signup", false},
		{"GET", "/-/signup", false},
		{"POST", "/-/signup", true},
		{"POST", "/-/login", false},
		{"PUT", "/pw-api/articles/x", true},
		{"DELETE", "/pw-api/files/3", true},
		{"POST", "/pw-api/modules", true},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, nil)
		if got := guarded(r); got != c.want {
			t.Errorf("guarded(%s %s) = %v, want %v", c.method, c.path, got, c.want)
		}
	}
}
