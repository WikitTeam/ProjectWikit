package modules

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

type forumData struct {
	module.Data
	categories []db.ForumCategory
	threads    []db.ForumThread
	first      map[int64]db.ForumThreadPost
	contents   map[int64]db.ForumPostContent
	counts     map[int64]int
	users      []db.User
	denied     bool

	askedIDs    []int64
	askedOffset int
	askedLimit  int
}

func (d *forumData) Subject(*db.User) (perms.Subject, error) {
	if d.denied {
		return perms.Subject{Anonymous: true}, nil
	}
	return perms.Subject{
		Anonymous: true,
		Roles:     []perms.Role{{Permissions: []string{perms.ViewForumCategories}}},
	}, nil
}

func (d *forumData) ForumCategories() ([]db.ForumCategory, error) { return d.categories, nil }

func (d *forumData) ForumThreadsInCategories(ids []int64, offset, limit int) ([]db.ForumThread, error) {
	d.askedIDs, d.askedOffset, d.askedLimit = ids, offset, limit
	return d.threads, nil
}

func (d *forumData) ForumFirstPosts([]int64) (map[int64]db.ForumThreadPost, error) {
	return d.first, nil
}

func (d *forumData) ForumThreadPostCounts([]int64) (map[int64]int, error) { return d.counts, nil }

func (d *forumData) ForumPostContents([]int64) (map[int64]db.ForumPostContent, error) {
	return d.contents, nil
}

func (d *forumData) UsersByIDs([]int64) ([]db.User, error) { return d.users, nil }

func noticeData() *forumData {
	author := int64(4)
	category := int64(88)
	return &forumData{
		categories: []db.ForumCategory{{ID: 88, Name: "Announcements"}},
		threads: []db.ForumThread{{
			ID: 7, Name: "本版规范", CategoryID: &category, AuthorID: &author,
			CreatedAt: time.Unix(1600000000, 0),
		}},
		first:    map[int64]db.ForumThreadPost{7: {ID: 30, ThreadID: 7}},
		contents: map[int64]db.ForumPostContent{30: {Source: "**rules** here"}},
		counts:   map[int64]int{7: 4},
		users:    []db.User{{ID: 4, Username: "kakushi"}},
	}
}

func renderedFrontForum(t *testing.T, data *forumData, params map[string]string, body string) string {
	t.Helper()
	env := module.Env{
		Data: data,
		Render: func(source string, _ *page.Context) (string, error) {
			return "<rendered>" + source + "</rendered>", nil
		},
	}
	out, err := module.Render(env, "frontforum", params, body)
	if err != nil {
		t.Fatalf("Render() err = %v, want nil", err)
	}
	return out
}

func TestFrontForumWrapsEachItem(t *testing.T) {
	got := renderedFrontForum(t, noticeData(), map[string]string{"category": "88"}, "%%title%%")

	want := `<div class="front-forum-box"><div><rendered>本版规范` + "\n" + `</rendered></div></div>`
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestFrontForumVars(t *testing.T) {
	tests := []struct{ body, want string }{
		{"%%title%%", "本版规范"},
		{"%%link%%", "/forum/t-7/"},
		{"%%linked_title%%", "[/forum/t-7/ 本版规范]"},
		{"%%title_linked%%", "[/forum/t-7/ 本版规范]"},
		{"%%author%%", "[[*user kakushi]]"},
		{"%%date%%", "[[date 1600000000]]"},
		{"%%date|%Y/%m/%d%%", `[[date 1600000000 format="%Y/%m/%d"]]`},
		{"%%comments%%", "3"},
		{"%%category%%", "[/forum/c-88/announcements Announcements]"},
		{"%%content%%", "**rules** here"},
		{"%%body%%", "**rules** here"},
	}
	for _, tt := range tests {
		got := renderedFrontForum(t, noticeData(), map[string]string{"category": "88"}, tt.body)
		if !strings.Contains(got, tt.want) {
			t.Errorf("Render(%q) = %q, want substring %q", tt.body, got, tt.want)
		}
	}
}

func TestFrontForumReadsSeveralCategories(t *testing.T) {
	data := noticeData()
	data.categories = append(data.categories, db.ForumCategory{ID: 99, Name: "Old"})
	renderedFrontForum(t, data, map[string]string{"category": "88;99;404"}, "%%title%%")

	want := []int64{88, 99}
	if len(data.askedIDs) != len(want) {
		t.Fatalf("askedIDs = %v, want %v", data.askedIDs, want)
	}
	for i := range want {
		if data.askedIDs[i] != want[i] {
			t.Errorf("askedIDs[%d] = %d, want %d", i, data.askedIDs[i], want[i])
		}
	}
}

func TestFrontForumWindow(t *testing.T) {
	data := noticeData()
	renderedFrontForum(t, data, map[string]string{"category": "88", "limit": "5", "offset": "2"}, "%%title%%")

	if data.askedLimit != 5 {
		t.Errorf("askedLimit = %d, want 5", data.askedLimit)
	}
	if data.askedOffset != 2 {
		t.Errorf("askedOffset = %d, want 2", data.askedOffset)
	}
}

func TestFrontForumDefaultsToTwenty(t *testing.T) {
	data := noticeData()
	renderedFrontForum(t, data, map[string]string{"category": "88"}, "%%title%%")

	if data.askedLimit != frontForumLimit {
		t.Errorf("askedLimit = %d, want %d", data.askedLimit, frontForumLimit)
	}
}

func TestFrontForumWithoutACategory(t *testing.T) {
	_, err := module.Render(module.Env{
		Data:   noticeData(),
		Render: func(string, *page.Context) (string, error) { return "", nil },
	}, "frontforum", map[string]string{}, "%%title%%")

	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) {
		t.Fatalf("Render() err = %v, want a module error", err)
	}
	if moduleErr.Message != "module-frontforum-no-category" {
		t.Errorf("Render() message = %q, want %q", moduleErr.Message, "module-frontforum-no-category")
	}
}

func TestFrontForumHidesEverythingWithoutPermission(t *testing.T) {
	data := noticeData()
	data.denied = true

	got := renderedFrontForum(t, data, map[string]string{"category": "88"}, "%%title%%")
	if want := `<div class="front-forum-box"></div>`; got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestFrontForumSkipsTheSummaryRenderWhenUnused(t *testing.T) {
	calls := 0
	env := module.Env{
		Data:              noticeData(),
		Render:            func(source string, _ *page.Context) (string, error) { return source, nil },
		RenderMessageText: func(string) (string, error) { calls++; return "plain", nil },
	}
	if _, err := module.Render(env, "frontforum", map[string]string{"category": "88"}, "%%title%%"); err != nil {
		t.Fatalf("Render() err = %v, want nil", err)
	}
	if calls != 0 {
		t.Errorf("RenderMessageText calls = %d, want 0", calls)
	}
}

func TestFrontForumSummary(t *testing.T) {
	env := module.Env{
		Data:              noticeData(),
		Render:            func(source string, _ *page.Context) (string, error) { return source, nil },
		RenderMessageText: func(string) (string, error) { return "  rules here  ", nil },
	}
	got, err := module.Render(env, "frontforum", map[string]string{"category": "88"}, "%%summary%%")
	if err != nil {
		t.Fatalf("Render() err = %v, want nil", err)
	}
	if !strings.Contains(got, "rules here") {
		t.Errorf("Render(\"%%%%summary%%%%\") = %q, want substring %q", got, "rules here")
	}
}

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		in    string
		limit int
		want  string
	}{
		{"abc", 5, "abc"},
		{"abcdef", 3, "abc"},
		{"本版规范", 2, "本版"},
	}
	for _, c := range cases {
		if got := truncateRunes(c.in, c.limit); got != c.want {
			t.Errorf("truncateRunes(%q, %d) = %q, want %q", c.in, c.limit, got, c.want)
		}
	}
}

func TestFrontForumCategoryIDs(t *testing.T) {
	cases := []struct {
		in   string
		want []int64
	}{
		{"88", []int64{88}},
		{" 88 ; 99 ", []int64{88, 99}},
		{"88;bad;0;-1", []int64{88}},
		{"", nil},
	}
	for _, c := range cases {
		got := frontForumCategoryIDs(c.in)
		if len(got) != len(c.want) {
			t.Errorf("frontForumCategoryIDs(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("frontForumCategoryIDs(%q)[%d] = %d, want %d", c.in, i, got[i], c.want[i])
			}
		}
	}
}
