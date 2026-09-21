package articlepage

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestUnwrapParagraphs(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<p>one line</p>", "one line"},
		{"<p>a</p>\n<span>b</span>\n<p>c</p>", "a\n<span>b</span>\nc"},
		{"", ""},
		{"no markup", "no markup"},
	}
	for _, c := range cases {
		if got := unwrapParagraphs(c.in); got != c.want {
			t.Errorf("unwrapParagraphs(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMissingNameResolves404PageName(t *testing.T) {
	resolve := missingName("component:no-such-page")

	got, ok := resolve("404_page_name")
	if !ok {
		t.Fatal("missingName(...)(\"404_page_name\") = _, false, want true")
	}
	if want := "component:no-such-page"; got != want {
		t.Errorf("missingName(...)(\"404_page_name\") = %q, want %q", got, want)
	}
}

func TestMissingNameLeavesOtherNames(t *testing.T) {
	resolve := missingName("no-such-page")

	for _, name := range []string{"fullname", "404_page_name ", "404_PAGE_NAME"} {
		if _, ok := resolve(name); ok {
			t.Errorf("missingName(...)(%q) = _, true, want false", name)
		}
	}
}

func TestContextCarriesTheCSRFToken(t *testing.T) {
	h := &Handler{}
	req := &request{article: &db.Article{Category: "_default", Name: "main"}, csrf: "a-token"}

	if got := h.context(req, req.article).CSRF; got != "a-token" {
		t.Errorf("context(...).CSRF = %q, want %q", got, "a-token")
	}
}
