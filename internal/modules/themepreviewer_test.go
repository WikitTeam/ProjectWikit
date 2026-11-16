package modules

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func TestThemePreviewerDropsSiteTheme(t *testing.T) {
	pc := page.NewContext(nil, nil, nil, nil)
	url := "https://bhl.scpwikicn.com/css/min/normalize.min.css"
	if _, err := renderThemePreviewer(module.Env{Page: pc}, map[string]string{"noUi": "true", "theme_url": url}, ""); err != nil {
		t.Fatalf("renderThemePreviewer() err = %v, want nil", err)
	}
	if pc.DropTheme != page.DropSiteTheme {
		t.Errorf("renderThemePreviewer() DropTheme = %d, want %d", pc.DropTheme, page.DropSiteTheme)
	}
	want := `@import url("` + url + `");`
	if len(pc.ComputedStyles) != 1 || pc.ComputedStyles[0] != want {
		t.Errorf("renderThemePreviewer() ComputedStyles = %q, want [%q]", pc.ComputedStyles, want)
	}
}

func TestThemePreviewerWithoutURL(t *testing.T) {
	pc := page.NewContext(nil, nil, nil, nil)
	if _, err := renderThemePreviewer(module.Env{Page: pc}, nil, ""); err != nil {
		t.Fatalf("renderThemePreviewer() err = %v, want nil", err)
	}
	if len(pc.ComputedStyles) != 0 {
		t.Errorf("renderThemePreviewer() ComputedStyles = %q, want none", pc.ComputedStyles)
	}
}

func TestStylesheetURL(t *testing.T) {
	cases := map[string]bool{
		"https://a.example/x.css":   true,
		"HTTP://a.example/x.css":    true,
		"//a.example/x.css":         true,
		"/local--files/x.css":       true,
		"javascript:alert(1)":       false,
		"x.css":                     false,
		`https://a.example/");x{}`:  false,
		"https://a.example/a b.css": false,
	}
	for in, want := range cases {
		if got := stylesheetURL(in); got != want {
			t.Errorf("stylesheetURL(%q) = %v, want %v", in, got, want)
		}
	}
}
