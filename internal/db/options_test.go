package db

import (
	"context"
	"errors"
	"testing"
)

func TestCommentInfoOfPageWithPosts(t *testing.T) {
	d := newTestDB(t)

	got, err := d.CommentInfo(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("CommentInfo() err = %v, want nil", err)
	}
	if got.ThreadID == 0 {
		t.Errorf("CommentInfo(probe:full).ThreadID = 0, want a thread id")
	}
	if got.Count != 2 {
		t.Errorf("CommentInfo(probe:full).Count = %d, want 2", got.Count)
	}
}

func TestCommentInfoOfPageWithoutThread(t *testing.T) {
	d := newTestDB(t)

	got, err := d.CommentInfo(context.Background(), 0)
	if err != nil {
		t.Fatalf("CommentInfo() err = %v, want nil", err)
	}
	if got != (CommentInfo{}) {
		t.Errorf("CommentInfo(0) = %+v, want the zero value", got)
	}
}

func TestCommentInfoOfPageWithoutPosts(t *testing.T) {
	d := newTestDB(t)

	got, err := d.CommentInfo(context.Background(), articleID(t, d, "probe:bare"))
	if err != nil {
		t.Fatalf("CommentInfo() err = %v, want nil", err)
	}
	if got.Count != 0 {
		t.Errorf("CommentInfo(probe:bare).Count = %d, want 0", got.Count)
	}
}

func TestSubscribedToArticle(t *testing.T) {
	d := newTestDB(t)
	author := userID(t, d, "probe-author")

	got, err := d.SubscribedToArticle(context.Background(), author, articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("SubscribedToArticle() err = %v, want nil", err)
	}
	if !got {
		t.Errorf("SubscribedToArticle(probe-author, probe:full) = false, want true")
	}
}

func TestSubscribedToArticleOfAnotherPage(t *testing.T) {
	d := newTestDB(t)
	author := userID(t, d, "probe-author")

	got, err := d.SubscribedToArticle(context.Background(), author, articleID(t, d, "probe:bare"))
	if err != nil {
		t.Fatalf("SubscribedToArticle() err = %v, want nil", err)
	}
	if got {
		t.Errorf("SubscribedToArticle(probe-author, probe:bare) = true, want false")
	}
}

func TestSubscribedToThread(t *testing.T) {
	d := newTestDB(t)
	author := userID(t, d, "probe-author")

	info, err := d.CommentInfo(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("CommentInfo() err = %v, want nil", err)
	}
	got, err := d.SubscribedToThread(context.Background(), author, info.ThreadID)
	if err != nil {
		t.Fatalf("SubscribedToThread() err = %v, want nil", err)
	}
	if !got {
		t.Errorf("SubscribedToThread(probe-author, %d) = false, want true", info.ThreadID)
	}
}

func TestUserPreferenceStoresPythonRepr(t *testing.T) {
	d := newTestDB(t)

	got, err := d.UserPreference(context.Background(), userID(t, d, "probe-author"),
		"qol", "advanced_source_editor_enabled")
	if err != nil {
		t.Fatalf("UserPreference() err = %v, want nil", err)
	}
	if got != "True" {
		t.Errorf("UserPreference(probe-author) = %q, want %q", got, "True")
	}
}

func TestUserPreferenceUnset(t *testing.T) {
	d := newTestDB(t)

	_, err := d.UserPreference(context.Background(), userID(t, d, "probevoter"),
		"qol", "advanced_source_editor_enabled")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("UserPreference(probevoter) err = %v, want ErrNotFound", err)
	}
}

func TestSiteCanCreateTagsReadsTheSiteRowAlone(t *testing.T) {
	d := newTestDB(t)

	site, err := d.SiteByHosts(context.Background(), []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts() err = %v, want nil", err)
	}
	got, err := d.SiteCanCreateTags(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("SiteCanCreateTags() err = %v, want nil", err)
	}
	if got != CreateTagsDefault {
		t.Errorf("SiteCanCreateTags() = %q, want %q", got, CreateTagsDefault)
	}
}
