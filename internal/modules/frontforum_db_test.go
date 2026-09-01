package modules

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

type forumLiveData struct {
	module.Data
	ctx context.Context
	d   *db.DB
}

func (l forumLiveData) Subject(*db.User) (perms.Subject, error) {
	return perms.Subject{
		Anonymous: true,
		Roles:     []perms.Role{{Permissions: []string{perms.ViewForumCategories}}},
	}, nil
}

func (l forumLiveData) ForumCategories() ([]db.ForumCategory, error) {
	return l.d.ForumCategories(l.ctx)
}

func (l forumLiveData) ForumThreadsInCategories(ids []int64, offset, limit int) ([]db.ForumThread, error) {
	return l.d.ForumThreadsInCategories(l.ctx, ids, offset, limit)
}

func (l forumLiveData) ForumFirstPosts(ids []int64) (map[int64]db.ForumThreadPost, error) {
	return l.d.ForumFirstPosts(l.ctx, ids)
}

func (l forumLiveData) ForumThreadPostCounts(ids []int64) (map[int64]int, error) {
	return l.d.ForumThreadPostCounts(l.ctx, ids)
}

func (l forumLiveData) ForumPostContents(ids []int64) (map[int64]db.ForumPostContent, error) {
	return l.d.ForumPostContents(l.ctx, ids)
}

func (l forumLiveData) UsersByIDs(ids []int64) ([]db.User, error) {
	return l.d.UsersByIDs(l.ctx, ids)
}

func forumLive(t *testing.T) forumLiveData {
	t.Helper()
	dsn := os.Getenv(db.EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", db.EnvDSN)
	}
	ctx := context.Background()
	d, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db.Open() err = %v, want nil", err)
	}
	t.Cleanup(d.Close)
	return forumLiveData{ctx: ctx, d: d}
}

func TestFrontForumOnALiveDatabase(t *testing.T) {
	data := forumLive(t)
	var items []string
	env := module.Env{
		Data: data,
		Render: func(source string, _ *page.Context) (string, error) {
			items = append(items, strings.TrimSpace(source))
			return source, nil
		},
	}

	if _, err := module.Render(env, "frontforum",
		map[string]string{"category": "55;61", "limit": "3"},
		"%%title%% | %%comments%% | %%category%% | %%link%%"); err != nil {
		t.Fatalf("Render() err = %v, want nil", err)
	}

	want := []string{
		"Probe Long Thread | 11 | [/forum/c-61/probe-talk Probe Talk] | /forum/t-122/probe-long-thread",
		"Probe Deep Thread | 5 | [/forum/c-61/probe-talk Probe Talk] | /forum/t-121/probe-deep-thread",
		"Probe Pinned Thread | 1 | [/forum/c-55/probe-chat Probe Chat] | /forum/t-99/probe-pinned-thread",
	}
	if len(items) != len(want) {
		t.Fatalf("rendered %d items, want %d", len(items), len(want))
	}
	for i := range want {
		if items[i] != want[i] {
			t.Errorf("item[%d] = %q, want %q", i, items[i], want[i])
		}
	}
}
