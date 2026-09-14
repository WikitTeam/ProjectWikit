package modules

import (
	"strings"
	"testing"
	"time"

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

func TestSearchExcerptStaysAroundTheWord(t *testing.T) {
	body := "title\n\n" + strings.Repeat("a", 300) + " needle " + strings.Repeat("b", 300)

	got := searchExcerpt(body, []string{"needle"})
	if !strings.Contains(got, "needle") {
		t.Errorf("searchExcerpt() = %q, want it to carry the word", got)
	}
	if runes := []rune(got); len(runes) > 2*searchExcerptPad+len("needle")+2 {
		t.Errorf("len(searchExcerpt()) = %d, want at most %d",
			len(runes), 2*searchExcerptPad+len("needle")+2)
	}
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "…") {
		t.Errorf("searchExcerpt() = %q, want it marked as cut on both sides", got)
	}
}

func TestSearchExcerptWithoutAMatch(t *testing.T) {
	body := "title\n\n" + strings.Repeat("c", 300)

	got := []rune(searchExcerpt(body, []string{"needle"}))
	if len(got) != searchExcerptLen+1 {
		t.Errorf("len(searchExcerpt(no match)) = %d, want %d", len(got), searchExcerptLen+1)
	}
}

func TestSearchExcerptCountsCharactersNotBytes(t *testing.T) {
	body := "标题\n\n" + strings.Repeat("文", 200) + "针" + strings.Repeat("字", 200)

	got := searchExcerpt(body, []string{"针"})
	if !strings.Contains(got, "针") {
		t.Errorf("searchExcerpt() = %q, want it to carry the word", got)
	}
	if runes := []rune(got); len(runes) > 2*searchExcerptPad+3 {
		t.Errorf("len(searchExcerpt()) = %d, want at most %d", len(runes), 2*searchExcerptPad+3)
	}
}

func TestSearchExcerptDropsTheHeading(t *testing.T) {
	if got := searchExcerpt("the title\n\nthe body", nil); got != "the body" {
		t.Errorf("searchExcerpt() = %q, want %q", got, "the body")
	}
}

func TestSearchTagsSplitsOnBothSeparators(t *testing.T) {
	got := searchTags(" alpha, beta  -gamma ")
	want := []string{"alpha", "beta", "-gamma"}
	if len(got) != len(want) {
		t.Fatalf("searchTags() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("searchTags()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSearchDate(t *testing.T) {
	from := searchDate("2024-03-04", false)
	if from == nil || from.Format(time.RFC3339) != "2024-03-04T00:00:00Z" {
		t.Errorf("searchDate(start) = %v, want 2024-03-04T00:00:00Z", from)
	}
	to := searchDate(" 2024-03-04 ", true)
	if to == nil || to.Format(time.RFC3339) != "2024-03-04T23:59:59Z" {
		t.Errorf("searchDate(end) = %v, want 2024-03-04T23:59:59Z", to)
	}
	for _, raw := range []string{"", "not a date", "2024-13-40"} {
		if got := searchDate(raw, false); got != nil {
			t.Errorf("searchDate(%q) = %v, want nil", raw, got)
		}
	}
}
