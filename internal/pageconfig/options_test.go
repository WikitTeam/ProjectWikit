package pageconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/article"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const (
	optionsGoldenPath = "testdata/options.golden"
	optionsCorpusPath = "testdata/options_corpus.json"
)

type permRoleSpec struct {
	Slug         string   `json:"slug"`
	Permissions  []string `json:"permissions"`
	Restrictions []string `json:"restrictions"`
}

type votesSpec struct {
	Rates []int `json:"rates"`
}

type optionsCase struct {
	Name          string         `json:"name"`
	PageID        string         `json:"page_id"`
	HasArticle    bool           `json:"has_article"`
	Anonymous     bool           `json:"anonymous"`
	Superuser     bool           `json:"superuser"`
	Inactive      bool           `json:"inactive"`
	Roles         []permRoleSpec `json:"roles"`
	Locked        bool           `json:"locked"`
	Author        bool           `json:"author"`
	RatingMode    string         `json:"rating_mode"`
	Votes         votesSpec      `json:"votes"`
	Path          string         `json:"path"`
	CommentCount  int            `json:"comment_count"`
	CommentThread int64          `json:"comment_thread_id"`
	CanCreateTags bool           `json:"can_create_tags"`
	IsWatching    bool           `json:"is_watching"`
	SourceEditor  bool           `json:"advanced_source_editor"`
}

func optionsCorpus() []optionsCase {
	reader := permRoleSpec{Slug: "everyone", Permissions: []string{perms.ViewArticles, perms.ViewArticleComments}}
	member := permRoleSpec{Slug: "registered", Permissions: []string{perms.CommentArticles, perms.RateArticles}}
	editor := permRoleSpec{
		Slug: "editor",
		Permissions: []string{
			perms.EditArticles, perms.TagArticles, perms.MoveArticles, perms.DeleteArticles,
			perms.LockArticles, perms.ManageArticleFiles, perms.ManageArticleAuthors,
			perms.ResetArticleVotes, perms.CreateArticles,
		},
	}

	single := func(name string) optionsCase {
		return optionsCase{
			Name: "only " + name, PageID: "main", HasArticle: true,
			Roles:      []permRoleSpec{{Slug: "everyone", Permissions: []string{name}}},
			RatingMode: page.RatingModeUpDown,
		}
	}
	granted := []string{
		perms.ViewArticles, perms.RateArticles, perms.CreateArticles, perms.EditArticles,
		perms.TagArticles, perms.MoveArticles, perms.LockArticles, perms.ManageArticleFiles,
		perms.DeleteArticles, perms.ResetArticleVotes, perms.CommentArticles,
		perms.ViewArticleComments, perms.ManageArticleAuthors,
	}

	cases := []optionsCase{
		{
			Name: "anonymous reader", PageID: "main", HasArticle: true, Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeUpDown,
			Votes: votesSpec{Rates: []int{1, 1, -1}},
		},
		{
			Name: "signed in member", PageID: "main", HasArticle: true,
			Roles: []permRoleSpec{reader, member}, RatingMode: page.RatingModeUpDown,
			Votes: votesSpec{Rates: []int{1, 1, -1}}, CommentCount: 4, CommentThread: 1201,
		},
		{
			Name: "editor", PageID: "scp-173", HasArticle: true,
			Roles: []permRoleSpec{reader, member, editor}, RatingMode: page.RatingModeUpDown,
			Votes: votesSpec{Rates: []int{1, 1, 1, -1}}, CommentCount: 1, CommentThread: 1202, CanCreateTags: true,
		},
		{
			Name: "superuser on a locked page", PageID: "main", HasArticle: true, Superuser: true,
			Roles: []permRoleSpec{reader}, Locked: true, RatingMode: page.RatingModeUpDown,
		},
		{
			Name: "author of a page", PageID: "main", HasArticle: true, Author: true,
			Roles: []permRoleSpec{reader, member}, RatingMode: page.RatingModeUpDown,
		},
		{
			Name: "inactive user", PageID: "main", HasArticle: true, Inactive: true,
			Roles: []permRoleSpec{reader, member}, RatingMode: page.RatingModeUpDown,
		},
		{
			Name: "missing page", PageID: "no-such-page", Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeDisabled,
		},
		{
			Name: "stars", PageID: "probestars:rated", HasArticle: true, Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeStars,
			Votes: votesSpec{Rates: []int{4, 5}},
		},
		{
			Name: "stars without votes", PageID: "probestars:rated", HasArticle: true, Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeStars,
		},
		{
			Name: "stars rounding to a whole number", PageID: "probestars:rated", HasArticle: true, Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeStars,
			Votes: votesSpec{Rates: []int{4, 4}},
		},
		{
			Name: "rating disabled", PageID: "main", HasArticle: true, Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeDisabled,
			Votes: votesSpec{Rates: []int{1, 1}},
		},
		{
			Name: "path params", PageID: "main", HasArticle: true, Anonymous: true,
			Roles: []permRoleSpec{reader}, RatingMode: page.RatingModeUpDown,
			Path: "main/offset/20/tag/中文/bare",
		},
		{
			Name: "watching", PageID: "main", HasArticle: true,
			Roles: []permRoleSpec{reader, member}, RatingMode: page.RatingModeUpDown,
			IsWatching: true,
		},
		{
			Name: "advanced source editor", PageID: "main", HasArticle: true,
			Roles: []permRoleSpec{reader, member}, RatingMode: page.RatingModeUpDown,
			SourceEditor: true,
		},
	}
	for _, name := range granted {
		cases = append(cases, single(name))
	}
	return cases
}

func toSubject(c optionsCase) perms.Subject {
	subject := perms.Subject{
		Anonymous: c.Anonymous,
		Active:    !c.Inactive,
		Superuser: c.Superuser,
	}
	for i, spec := range c.Roles {
		subject.Roles = append(subject.Roles, perms.Role{
			ID:           int64(i + 1),
			Permissions:  spec.Permissions,
			Restrictions: spec.Restrictions,
		})
	}
	return subject
}

func toStats(spec votesSpec) db.VoteStats {
	stats := db.VoteStats{Count: len(spec.Rates)}
	for _, rate := range spec.Rates {
		stats.Sum += float64(rate)
		if rate == 1 {
			stats.GoodUpDown++
		}
		if rate >= 3 {
			stats.GoodStars++
		}
	}
	if stats.Count > 0 {
		stats.Average = stats.Sum / float64(stats.Count)
	}
	return stats
}

func toOptions(c optionsCase) Options {
	var object *perms.Object
	if c.HasArticle {
		object = &perms.Object{Locked: c.Locked, Author: c.Author}
	}
	_, params := article.ParsePath(c.Path, "main")
	if c.Path == "" {
		params = nil
	}
	rating := page.DisabledRating()
	if c.HasArticle {
		rating = page.RatingOf(c.RatingMode, toStats(c.Votes))
	}
	return Options{
		PageID:          c.PageID,
		NormalizedName:  c.PageID,
		HasArticle:      c.HasArticle,
		Anonymous:       c.Anonymous,
		Perms:           perms.Resolve(toSubject(c), object),
		Rating:          rating,
		PathParams:      params,
		CommentCount:    c.CommentCount,
		CommentThreadID: c.CommentThread,
		CommentSlug:     c.PageID,
		CanCreateTags:   c.CanCreateTags,
		IsWatching:      c.IsWatching,
		Preferences:     Preferences{AdvancedSourceEditor: c.SourceEditor},
	}
}

func TestOptionsMatchesGolden(t *testing.T) {
	cases := optionsCorpus()

	var b strings.Builder
	for _, c := range cases {
		got, err := toOptions(c).JSON()
		if err != nil {
			t.Fatalf("Options(%s).JSON() err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, got)
	}
	got := b.String()

	if *update {
		writeOptionsCorpus(t, cases)
		if err := os.WriteFile(filepath.FromSlash(optionsGoldenPath), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", optionsGoldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(filepath.FromSlash(optionsGoldenPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", optionsGoldenPath, err)
	}
	if got != string(want) {
		t.Errorf("Options(corpus) = %q, want %q", got, string(want))
	}
}

func writeOptionsCorpus(t *testing.T, cases []optionsCase) {
	t.Helper()
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(corpus) err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.FromSlash(optionsCorpusPath), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", optionsCorpusPath, err)
	}
}

func TestOptionsOfMissingPageHasNoCommentThread(t *testing.T) {
	got, err := Options{PageID: "gone", NormalizedName: "gone"}.JSON()
	if err != nil {
		t.Fatalf("JSON() err = %v, want nil", err)
	}
	if !strings.Contains(got, `"commentThread": null`) {
		t.Errorf("Options(missing page) = %q, want it to carry \"commentThread\": null", got)
	}
}
