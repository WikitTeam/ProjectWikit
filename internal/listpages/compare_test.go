package listpages

import "testing"

func TestCompareFormNumbers(t *testing.T) {
	cases := []struct {
		value, op, want string
		match           bool
	}{
		{"10", ">", "9", true},
		{"9", ">", "10", false},
		{"2.5", "<=", "2.5", true},
		{"-1", "<", "0", true},
		{"b", ">", "a", true},
		{"inf", ">", "1", true},
		{"10", "<", "9a", true},
	}
	for _, c := range cases {
		if got := compareForm(c.value, c.op, c.want); got != c.match {
			t.Errorf("compareForm(%q, %q, %q) = %v, want %v", c.value, c.op, c.want, got, c.match)
		}
	}
}
