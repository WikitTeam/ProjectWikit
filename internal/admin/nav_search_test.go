package admin

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
)

func TestScreenTemplatesCoverTheMenu(t *testing.T) {
	for _, g := range groups {
		for _, e := range g.entries {
			names, ok := screenTemplates[e.slug]
			if !ok {
				t.Errorf("screenTemplates[%q] missing, want the templates it renders", e.slug)
				continue
			}
			for _, name := range names {
				if _, err := fs.Stat(files, "templates/"+name); err != nil {
					t.Errorf("screenTemplates[%q] names %q, which does not exist", e.slug, name)
				}
			}
		}
	}
}

func TestSearchTextFindsFieldsOnTheScreen(t *testing.T) {
	bundle, err := i18n.LoadKeys()
	if err != nil {
		t.Fatalf("LoadKeys() err = %v, want nil", err)
	}
	loc := bundle.Localizer("zh-hans")
	got := searchText(loc, siteSlug)
	for _, want := range []string{loc.T("admin.f-self-votes"), loc.T("admin.f-media-domain")} {
		if !strings.Contains(got, want) {
			t.Errorf("searchText(site) = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, loc.T("admin.h-media-domain")) {
		t.Errorf("searchText(site) = %q, want no hints", got)
	}
}
