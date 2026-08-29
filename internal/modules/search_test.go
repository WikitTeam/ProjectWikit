package modules

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func searchEnv(path page.PathParams) module.Env {
	return module.Env{Page: page.NewContext(&db.Article{ID: 1, Name: "site", Category: "search"}, nil, path, nil)}
}

func searchConfig(t *testing.T, path page.PathParams, params map[string]string) string {
	t.Helper()
	got, err := renderSearch(searchEnv(path), params, "")
	if err != nil {
		t.Fatalf("renderSearch() err = %v, want nil", err)
	}
	return got
}

func TestRenderSearchFallsBackToTheDefaultPlaceholder(t *testing.T) {
	got := searchConfig(t, nil, nil)
	want := `&quot;placeholder&quot;: &quot;module-search-placeholder&quot;`
	if !strings.Contains(got, want) {
		t.Errorf("renderSearch() = %q, want it to contain %q", got, want)
	}
}

func TestRenderSearchKeepsTheGivenPlaceholder(t *testing.T) {
	got := searchConfig(t, nil, map[string]string{"placeholder": "find"})
	if want := `&quot;placeholder&quot;: &quot;find&quot;`; !strings.Contains(got, want) {
		t.Errorf("renderSearch() = %q, want it to contain %q", got, want)
	}
}

func TestRenderSearchPrefersThePathQuery(t *testing.T) {
	got := searchConfig(t, page.PathParams{{Key: "q", Value: "from-path"}}, map[string]string{"q": "from-module"})
	if want := `&quot;q&quot;: &quot;from-path&quot;`; !strings.Contains(got, want) {
		t.Errorf("renderSearch() = %q, want it to contain %q", got, want)
	}
}

func TestRenderSearchTrimsThePathValues(t *testing.T) {
	got := searchConfig(t, page.PathParams{{Key: "author", Value: "  probe-author  "}}, nil)
	if want := `&quot;author&quot;: &quot;probe-author&quot;`; !strings.Contains(got, want) {
		t.Errorf("renderSearch() = %q, want it to contain %q", got, want)
	}
}

func TestRenderSearchLeavesTheModuleParamsUntrimmed(t *testing.T) {
	got := searchConfig(t, nil, map[string]string{"tags": " a b "})
	if want := `&quot;tags&quot;: &quot; a b &quot;`; !strings.Contains(got, want) {
		t.Errorf("renderSearch() = %q, want it to contain %q", got, want)
	}
}
