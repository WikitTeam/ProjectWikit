package page

import "testing"

func TestGalleries(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"[[gallery]]", `[[module Gallery]]`},
		{`[[gallery size="medium"]]`, `[[module Gallery size="medium"]]`},
		{"[[GALLERY]]", `[[module Gallery]]`},
		{"before [[gallery]] after", `before [[module Gallery]] after`},
		{"[[galleryish]]", "[[galleryish]]"},
		{"[[gallery-of-x]]", "[[gallery-of-x]]"},
		{"[[galleries size=1]]", "[[galleries size=1]]"},
		{`[[module Gallery size="medium"]]`, `[[module Gallery size="medium"]]`},
	}
	for _, c := range cases {
		if got := Galleries(c.in); got != c.want {
			t.Errorf("Galleries(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGalleriesLeavesAHeadCarryingABracket(t *testing.T) {
	in := `[[gallery size="a]b"]]`
	if got := Galleries(in); got != in {
		t.Errorf("Galleries(%q) = %q, want it left alone", in, got)
	}
}
