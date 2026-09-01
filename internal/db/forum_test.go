package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func forumDB(t *testing.T) (*DB, context.Context) {
	t.Helper()
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	ctx := context.Background()
	d, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	t.Cleanup(d.Close)
	return d, ctx
}

func TestForumSectionsComeBackInOrder(t *testing.T) {
	d, ctx := forumDB(t)
	sections, err := d.ForumSections(ctx)
	if err != nil {
		t.Fatalf("ForumSections() err = %v, want nil", err)
	}

	want := []struct {
		name           string
		hidden         bool
		hiddenForUsers bool
	}{
		{"Probe Open", false, false},
		{"Probe Hidden", true, false},
		{"Probe Staff", false, true},
	}
	if len(sections) != len(want) {
		t.Fatalf("len(ForumSections()) = %d, want %d", len(sections), len(want))
	}
	for i, w := range want {
		if sections[i].Name != w.name {
			t.Errorf("ForumSections()[%d].Name = %q, want %q", i, sections[i].Name, w.name)
		}
		if sections[i].IsHidden != w.hidden {
			t.Errorf("ForumSections()[%d].IsHidden = %t, want %t", i, sections[i].IsHidden, w.hidden)
		}
		if sections[i].IsHiddenForUsers != w.hiddenForUsers {
			t.Errorf("ForumSections()[%d].IsHiddenForUsers = %t, want %t", i, sections[i].IsHiddenForUsers, w.hiddenForUsers)
		}
	}
}

func TestForumSectionByID(t *testing.T) {
	d, ctx := forumDB(t)
	sections, err := d.ForumSections(ctx)
	if err != nil {
		t.Fatalf("ForumSections() err = %v, want nil", err)
	}
	got, err := d.ForumSection(ctx, sections[0].ID)
	if err != nil {
		t.Fatalf("ForumSection(%d) err = %v, want nil", sections[0].ID, err)
	}
	if got.Name != sections[0].Name {
		t.Errorf("ForumSection(%d).Name = %q, want %q", sections[0].ID, got.Name, sections[0].Name)
	}
}

func TestForumSectionOfAnUnknownID(t *testing.T) {
	d, ctx := forumDB(t)
	if _, err := d.ForumSection(ctx, -1); !errors.Is(err, ErrNotFound) {
		t.Errorf("ForumSection(-1) err = %v, want ErrNotFound", err)
	}
}

func TestForumCategoriesComeBackInOrder(t *testing.T) {
	d, ctx := forumDB(t)
	categories, err := d.ForumCategories(ctx)
	if err != nil {
		t.Fatalf("ForumCategories() err = %v, want nil", err)
	}

	want := []string{"Probe Chat", "Probe Hidden Chat", "Probe Staff Chat", "Probe Comments", "Probe Quiet", "Probe Busy", "Probe Talk"}
	if len(categories) != len(want) {
		t.Fatalf("len(ForumCategories()) = %d, want %d", len(categories), len(want))
	}
	for i, name := range want {
		if categories[i].Name != name {
			t.Errorf("ForumCategories()[%d].Name = %q, want %q", i, categories[i].Name, name)
		}
	}
}

func categoryNamed(t *testing.T, d *DB, ctx context.Context, name string) ForumCategory {
	t.Helper()
	categories, err := d.ForumCategories(ctx)
	if err != nil {
		t.Fatalf("ForumCategories() err = %v, want nil", err)
	}
	for _, c := range categories {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("ForumCategories() has no %q, want it; run oracle_seed.py first", name)
	return ForumCategory{}
}

func TestForumCategoryCounts(t *testing.T) {
	d, ctx := forumDB(t)
	chat := categoryNamed(t, d, ctx, "Probe Chat")
	got, err := d.ForumCategoryCounts(ctx, chat.ID)
	if err != nil {
		t.Fatalf("ForumCategoryCounts(chat) err = %v, want nil", err)
	}
	if got.Threads != 3 {
		t.Errorf("ForumCategoryCounts(chat).Threads = %d, want %d", got.Threads, 3)
	}
	if got.Posts != 6 {
		t.Errorf("ForumCategoryCounts(chat).Posts = %d, want %d", got.Posts, 6)
	}
}

func TestForumCategoryCountsOfAnEmptyCategory(t *testing.T) {
	d, ctx := forumDB(t)
	quiet := categoryNamed(t, d, ctx, "Probe Quiet")
	got, err := d.ForumCategoryCounts(ctx, quiet.ID)
	if err != nil {
		t.Fatalf("ForumCategoryCounts(quiet) err = %v, want nil", err)
	}
	if got.Threads != 0 || got.Posts != 0 {
		t.Errorf("ForumCategoryCounts(quiet) = %+v, want zeroes", got)
	}
}

func TestForumCategoryLastPost(t *testing.T) {
	d, ctx := forumDB(t)
	chat := categoryNamed(t, d, ctx, "Probe Chat")
	got, err := d.ForumCategoryLastPost(ctx, chat.ID)
	if err != nil {
		t.Fatalf("ForumCategoryLastPost(chat) err = %v, want nil", err)
	}
	if got.ThreadName != "Probe Pinned Thread" {
		t.Errorf("ForumCategoryLastPost(chat).ThreadName = %q, want %q", got.ThreadName, "Probe Pinned Thread")
	}
	want := time.Date(2023, 9, 10, 11, 17, 13, 0, time.UTC)
	if !got.CreatedAt.Equal(want) {
		t.Errorf("ForumCategoryLastPost(chat).CreatedAt = %v, want %v", got.CreatedAt.UTC(), want)
	}
	if got.ThreadArticleID != nil {
		t.Errorf("ForumCategoryLastPost(chat).ThreadArticleID = %v, want nil", *got.ThreadArticleID)
	}
	if got.AuthorID == nil {
		t.Error("ForumCategoryLastPost(chat).AuthorID = nil, want an id")
	}
}

func TestForumCategoryLastPostOfAnEmptyCategory(t *testing.T) {
	d, ctx := forumDB(t)
	quiet := categoryNamed(t, d, ctx, "Probe Quiet")
	if _, err := d.ForumCategoryLastPost(ctx, quiet.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("ForumCategoryLastPost(quiet) err = %v, want ErrNotFound", err)
	}
}

func TestForumCommentCountsReachEveryArticle(t *testing.T) {
	d, ctx := forumDB(t)
	comments := categoryNamed(t, d, ctx, "Probe Comments")
	byCategory, err := d.ForumCategoryCounts(ctx, comments.ID)
	if err != nil {
		t.Fatalf("ForumCategoryCounts(comments) err = %v, want nil", err)
	}
	if byCategory.Threads != 0 || byCategory.Posts != 0 {
		t.Errorf("ForumCategoryCounts(comments) = %+v, want zeroes", byCategory)
	}

	got, err := d.ForumCommentCounts(ctx)
	if err != nil {
		t.Fatalf("ForumCommentCounts() err = %v, want nil", err)
	}
	if got.Threads < 1 {
		t.Errorf("ForumCommentCounts().Threads = %d, want at least 1", got.Threads)
	}
	if got.Posts != 4 {
		t.Errorf("ForumCommentCounts().Posts = %d, want %d", got.Posts, 4)
	}
}

func TestForumCommentLastPostComesFromAnArticleThread(t *testing.T) {
	d, ctx := forumDB(t)
	got, err := d.ForumCommentLastPost(ctx)
	if err != nil {
		t.Fatalf("ForumCommentLastPost() err = %v, want nil", err)
	}
	if got.ThreadArticleID == nil {
		t.Error("ForumCommentLastPost().ThreadArticleID = nil, want an id")
	}
	if got.ThreadName != "Probe Full" {
		t.Errorf("ForumCommentLastPost().ThreadName = %q, want %q", got.ThreadName, "Probe Full")
	}
}

func TestUserByIDRoundTrips(t *testing.T) {
	d, ctx := forumDB(t)
	want, err := d.UserByUsername(ctx, "probe-author")
	if err != nil {
		t.Fatalf("UserByUsername(probe-author) err = %v, want nil", err)
	}
	got, err := d.UserByID(ctx, want.ID)
	if err != nil {
		t.Fatalf("UserByID(%d) err = %v, want nil", want.ID, err)
	}
	if got.Username != want.Username {
		t.Errorf("UserByID(%d).Username = %q, want %q", want.ID, got.Username, want.Username)
	}
}

func TestUserByIDOfAnUnknownID(t *testing.T) {
	d, ctx := forumDB(t)
	if _, err := d.UserByID(ctx, -1); !errors.Is(err, ErrNotFound) {
		t.Errorf("UserByID(-1) err = %v, want ErrNotFound", err)
	}
}

func TestForumThreadsInCategoriesKeepDateOrderAcrossCategories(t *testing.T) {
	d, ctx := forumDB(t)
	threads, err := d.ForumThreadsInCategories(ctx, []int64{55, 61}, 0, 20)
	if err != nil {
		t.Fatalf("ForumThreadsInCategories() err = %v, want nil", err)
	}

	want := []string{"Probe Long Thread", "Probe Deep Thread", "Probe Pinned Thread", "Probe Locked Thread", "Probe Thread"}
	if len(threads) != len(want) {
		t.Fatalf("len(ForumThreadsInCategories()) = %d, want %d", len(threads), len(want))
	}
	for i, name := range want {
		if threads[i].Name != name {
			t.Errorf("ForumThreadsInCategories()[%d].Name = %q, want %q", i, threads[i].Name, name)
		}
	}
}

func TestForumThreadsInCategoriesWindow(t *testing.T) {
	d, ctx := forumDB(t)
	threads, err := d.ForumThreadsInCategories(ctx, []int64{55, 61}, 2, 2)
	if err != nil {
		t.Fatalf("ForumThreadsInCategories() err = %v, want nil", err)
	}

	want := []string{"Probe Pinned Thread", "Probe Locked Thread"}
	if len(threads) != len(want) {
		t.Fatalf("len(ForumThreadsInCategories()) = %d, want %d", len(threads), len(want))
	}
	for i, name := range want {
		if threads[i].Name != name {
			t.Errorf("ForumThreadsInCategories()[%d].Name = %q, want %q", i, threads[i].Name, name)
		}
	}
}

func TestForumFirstPostsPickTheOldestRootPost(t *testing.T) {
	d, ctx := forumDB(t)
	first, err := d.ForumFirstPosts(ctx, []int64{97, 122})
	if err != nil {
		t.Fatalf("ForumFirstPosts() err = %v, want nil", err)
	}

	want := map[int64]int64{97: 22, 122: 54}
	if len(first) != len(want) {
		t.Fatalf("len(ForumFirstPosts()) = %d, want %d", len(first), len(want))
	}
	for thread, post := range want {
		if first[thread].ID != post {
			t.Errorf("ForumFirstPosts()[%d].ID = %d, want %d", thread, first[thread].ID, post)
		}
	}
}

func TestForumThreadPostCounts(t *testing.T) {
	d, ctx := forumDB(t)
	counts, err := d.ForumThreadPostCounts(ctx, []int64{97, 122})
	if err != nil {
		t.Fatalf("ForumThreadPostCounts() err = %v, want nil", err)
	}

	want := map[int64]int{97: 3, 122: 12}
	for thread, count := range want {
		if counts[thread] != count {
			t.Errorf("ForumThreadPostCounts()[%d] = %d, want %d", thread, counts[thread], count)
		}
	}
}
