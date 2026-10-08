package main

import "testing"

func TestDataDirArg(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"merge", "-from", "a", "-into", "b"}, ""},
		{[]string{"prune", "-data-dir", "/srv/wiki"}, "/srv/wiki"},
		{[]string{"create", "--data-dir", "/srv/wiki"}, "/srv/wiki"},
		{[]string{"-data-dir=/srv/wiki", "x"}, "/srv/wiki"},
		{[]string{"--data-dir=/srv/wiki"}, "/srv/wiki"},
		{[]string{"--", "-data-dir", "/srv/wiki"}, ""},
		{[]string{"-data-dir"}, ""},
	}
	for _, c := range cases {
		if got := dataDirArg(c.args); got != c.want {
			t.Errorf("dataDirArg(%q) = %q, want %q", c.args, got, c.want)
		}
	}
}
