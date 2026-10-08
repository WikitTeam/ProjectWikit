package admin

import "testing"

func TestWorthTelling(t *testing.T) {
	author, staff := int64(7), int64(9)
	pending := handled{status: "pending"}
	cases := []struct {
		name   string
		to     *int64
		by     int64
		before handled
		after  handled
		want   bool
	}{
		{"approved", &author, staff, pending, handled{status: "approved"}, true},
		{"still pending", &author, staff, pending, handled{status: "pending", reply: "soon"}, false},
		{"reply added later", &author, staff, handled{status: "approved"}, handled{status: "approved", reply: "welcome"}, true},
		{"saved unchanged", &author, staff, handled{status: "approved", reply: "x"}, handled{status: "approved", reply: "x"}, false},
		{"own ticket", &staff, staff, pending, handled{status: "closed"}, false},
		{"author deleted", nil, staff, pending, handled{status: "closed"}, false},
	}
	for _, c := range cases {
		if got := worthTelling(c.to, c.by, c.before, c.after, "pending"); got != c.want {
			t.Errorf("worthTelling(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}
