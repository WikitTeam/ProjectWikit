package modules

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func TestFeedSize(t *testing.T) {
	cases := []struct {
		params map[string]string
		want   int
	}{
		{map[string]string{}, 20},
		{map[string]string{"limit": "5"}, 5},
		{map[string]string{"perpage": "10", "limit": "50"}, 10},
		{map[string]string{"rsslimit": "30", "limit": "5"}, 30},
		{map[string]string{"rsslimit": "9999"}, 250},
	}
	for _, c := range cases {
		if got := feedSize(c.params); got != c.want {
			t.Errorf("feedSize(%v) = %d, want %d", c.params, got, c.want)
		}
	}
}

func TestFeedLink(t *testing.T) {
	pc := page.NewContext(&db.Article{Category: db.DefaultCategory, Name: "news"}, nil, nil, nil)
	if got := feedLink(pc, map[string]string{"category": "blog"}); got != "" {
		t.Errorf("feedLink(no rss) = %q, want empty", got)
	}
	got := feedLink(pc, map[string]string{"category": "blog", "rss": "Blog & news"})
	for _, want := range []string{`href="/pw-api/feed/listpages?category=blog&amp;page=news&amp;rss=Blog+%26+news"`, "Blog &amp; news"} {
		if !strings.Contains(got, want) {
			t.Errorf("feedLink() = %q, want %q", got, want)
		}
	}
}
