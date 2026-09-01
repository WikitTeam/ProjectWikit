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

func TestButtonsGathersBareArguments(t *testing.T) {
	cases := []struct{ in, want string }{
		{"[[button set-tags -a +b]]", `[[module Button type="set-tags" tags="-a +b"]]`},
		{"[[button set-tags -发现 +丢失]]", `[[module Button type="set-tags" tags="-发现 +丢失"]]`},
		{`[[button set-tags -a +b text="x"]]`, `[[module Button type="set-tags" tags="-a +b" text="x"]]`},
		{`[[button set-tags text="x" -a +b]]`, `[[module Button type="set-tags" tags="-a +b" text="x"]]`},
		{`[[button set-tags -a text="丢失 "]]`, `[[module Button type="set-tags" tags="-a" text="丢失 "]]`},
		{`[[button set-tags -a  +b]]`, `[[module Button type="set-tags" tags="-a +b"]]`},
	}
	for _, c := range cases {
		if got := Buttons(c.in); got != c.want {
			t.Errorf("Buttons(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestButtonsDropsAQuoteFromABareArgument(t *testing.T) {
	in := `[[button set-tags +a"b]]`
	want := `[[module Button type="set-tags" tags="+ab"]]`
	if got := Buttons(in); got != want {
		t.Errorf("Buttons(%q) = %q, want %q", in, got, want)
	}
}

func TestButtonsKeepsAnArgumentWithoutQuotes(t *testing.T) {
	in := `[[button set-tags -a text=x]]`
	want := `[[module Button type="set-tags" tags="-a" text=x]]`
	if got := Buttons(in); got != want {
		t.Errorf("Buttons(%q) = %q, want %q", in, got, want)
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
