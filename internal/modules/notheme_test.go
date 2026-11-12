package modules

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func TestNoThemeScope(t *testing.T) {
	cases := []struct {
		scope string
		want  page.ThemeDrop
	}{
		{"", page.DropSiteTheme},
		{"site", page.DropSiteTheme},
		{"ALL", page.DropAllThemes},
	}
	for _, c := range cases {
		pc := page.NewContext(nil, nil, nil, nil)
		if _, err := renderNoTheme(module.Env{Page: pc}, map[string]string{"scope": c.scope}, ""); err != nil {
			t.Fatalf("renderNoTheme(scope=%q) err = %v, want nil", c.scope, err)
		}
		if pc.DropTheme != c.want {
			t.Errorf("renderNoTheme(scope=%q) DropTheme = %d, want %d", c.scope, pc.DropTheme, c.want)
		}
	}
}

func TestNoThemeKeepsTheWiderDrop(t *testing.T) {
	pc := page.NewContext(nil, nil, nil, nil)
	pc.DropTheme = page.DropAllThemes
	if _, err := renderNoTheme(module.Env{Page: pc}, nil, ""); err != nil {
		t.Fatalf("renderNoTheme() err = %v, want nil", err)
	}
	if pc.DropTheme != page.DropAllThemes {
		t.Errorf("renderNoTheme() DropTheme = %d, want %d", pc.DropTheme, page.DropAllThemes)
	}
}

func TestNoThemeRejectsUnknownScope(t *testing.T) {
	if _, err := renderNoTheme(module.Env{}, map[string]string{"scope": "base"}, ""); err == nil {
		t.Errorf("renderNoTheme(scope=base) err = nil, want an error")
	}
}

func TestMergeKeepsDropTheme(t *testing.T) {
	outer := page.NewContext(nil, nil, nil, nil)
	inner := page.NewContext(nil, nil, nil, nil)
	inner.DropTheme = page.DropSiteTheme
	outer.Merge(inner)
	if outer.DropTheme != page.DropSiteTheme {
		t.Errorf("Merge() DropTheme = %d, want %d", outer.DropTheme, page.DropSiteTheme)
	}
}
