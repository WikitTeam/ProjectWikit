package page

import "testing"

func TestButtons(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"[[button edit]]", `[[module Button type="edit"]]`},
		{`[[button edit text="创建此页"]]`, `[[module Button type="edit" text="创建此页"]]`},
		{"[[button EDIT]]", `[[module Button type="EDIT"]]`},
		{"[[BUTTON edit]]", `[[module Button type="edit"]]`},
		{"before [[button edit]] after", `before [[module Button type="edit"]] after`},
		{"[[button edit]][[button edit]]", `[[module Button type="edit"]][[module Button type="edit"]]`},
		{"[[button]]", "[[button]]"},
		{"[[button ]]", "[[button ]]"},
		{"[[button 1edit]]", "[[button 1edit]]"},
		{"[[buttons edit]]", "[[buttons edit]]"},
		{"[[module Button type=\"edit\"]]", "[[module Button type=\"edit\"]]"},
	}
	for _, c := range cases {
		if got := Buttons(c.in); got != c.want {
			t.Errorf("Buttons(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestButtonsKeepsTheRestOfTheHead(t *testing.T) {
	in := `[[button edit text="a" class="b"]]`
	want := `[[module Button type="edit" text="a" class="b"]]`
	if got := Buttons(in); got != want {
		t.Errorf("Buttons(%q) = %q, want %q", in, got, want)
	}
}

func TestButtonsLeavesAHeadCarryingABracket(t *testing.T) {
	in := `[[button edit text="a]b"]]`
	if got := Buttons(in); got != in {
		t.Errorf("Buttons(%q) = %q, want it left alone", in, got)
	}
}
