package modules

import (
	"slices"
	"testing"
)

func TestSplitSigned(t *testing.T) {
	cases := []struct {
		raw        string
		tags       bool
		keep, skip []string
	}{
		{"scp tale -archived", false, []string{"scp", "tale"}, []string{"archived"}},
		{"+原创, -搞笑", true, []string{"原创"}, []string{"搞笑"}},
		{"- +", false, nil, nil},
		{"", true, nil, nil},
	}
	for _, c := range cases {
		keep, skip := splitSigned(c.raw, c.tags)
		if !slices.Equal(keep, c.keep) {
			t.Errorf("splitSigned(%q) keep = %q, want %q", c.raw, keep, c.keep)
		}
		if !slices.Equal(skip, c.skip) {
			t.Errorf("splitSigned(%q) skip = %q, want %q", c.raw, skip, c.skip)
		}
	}
}
