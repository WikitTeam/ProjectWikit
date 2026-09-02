package userpage

import "testing"

func TestNumericIDTakesTheDjangoShape(t *testing.T) {
	cases := []struct {
		name string
		id   int64
		ok   bool
	}{
		{"1-admin", 1, true},
		{"12-probe_author-2", 12, true},
		{"1", 0, false},
		{"1-", 0, false},
		{"1-中文", 0, false},
		{"1-a.b", 0, false},
		{"admin", 0, false},
		{"-1-admin", 0, false},
	}
	for _, c := range cases {
		id, ok := numericID(c.name)
		if ok != c.ok || id != c.id {
			t.Errorf("numericID(%q) = %d, %v, want %d, %v", c.name, id, ok, c.id, c.ok)
		}
	}
}
