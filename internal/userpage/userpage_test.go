package userpage

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestNumericIDTakesTheDjangoShape(t *testing.T) {
	cases := []struct {
		name string
		id   int64
		ok   bool
	}{
		{"1-admin", 1, true},
		{"12-probe_author-2", 12, true},
		{"1", 0, false},
		{"1-", 0, false},
		{"1-中文", 0, false},
		{"1-a.b", 0, false},
		{"admin", 0, false},
		{"-1-admin", 0, false},
	}
	for _, c := range cases {
		id, ok := numericID(c.name)
		if ok != c.ok || id != c.id {
			t.Errorf("numericID(%q) = %d, %v, want %d, %v", c.name, id, ok, c.id, c.ok)
		}
	}
}

func TestThreadNamePrefersTheArticleTitle(t *testing.T) {
	title, name, category := "SCP-173", "scp-173", "_default"
	post := db.UserPost{ThreadName: "Comments for scp-173",
		ArticleTitle: &title, ArticleName: &name, ArticleCategory: &category}
	if got := threadName(post); got != "SCP-173" {
		t.Errorf("threadName(comment thread) = %q, want %q", got, "SCP-173")
	}
}

func TestThreadNameKeepsTheThreadNameOutsideComments(t *testing.T) {
	id := int64(4)
	post := db.UserPost{ThreadName: "Announcements", ThreadCategoryID: &id}
	if got := threadName(post); got != "Announcements" {
		t.Errorf("threadName(forum thread) = %q, want %q", got, "Announcements")
	}
}

func TestThreadURL(t *testing.T) {
	if got, want := threadURL(2, "Probe Thread"), "/forum/t-2/probe-thread"; got != want {
		t.Errorf("threadURL(2, %q) = %q, want %q", "Probe Thread", got, want)
	}
}

func TestSplitFullName(t *testing.T) {
	cases := []struct{ in, first, last string }{
		{"Ada Lovelace", "Ada", "Lovelace"},
		{"Jean Luc Picard", "Jean Luc", "Picard"},
		{"Cher", "Cher", ""},
		{"  ", "", ""},
	}
	for _, c := range cases {
		first, last := splitFullName(c.in)
		if first != c.first || last != c.last {
			t.Errorf("splitFullName(%q) = %q, %q, want %q, %q", c.in, first, last, c.first, c.last)
		}
	}
}

func TestMediaTypeDropsTheCharset(t *testing.T) {
	if got, want := mediaType("text/plain; charset=utf-8"), "text/plain"; got != want {
		t.Errorf("mediaType(%q) = %q, want %q", "text/plain; charset=utf-8", got, want)
	}
}

func TestAnswers(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/-/favourites", true},
		{"/-/ratings", true},
		{"/-/liked-posts", true},
		{"/-/notifications", true},
		{"/-/notifications/all", true},
		{"/-/notifications/unread", true},
		{"/-/notifications/other", false},
		{"/-/messages", true},
		{"/-/messages/12", true},
		{"/-/messages/", false},
		{"/-/messages/bob", false},
		{"/-/messages/12/34", false},
		{"/-/login", false},
	}
	for _, c := range cases {
		if got := answers(c.path); got != c.want {
			t.Errorf("answers(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
