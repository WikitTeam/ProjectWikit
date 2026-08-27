package db

import (
	"context"
	"testing"
)

func articleID(t *testing.T, d *DB, ref string) int64 {
	t.Helper()
	a, err := d.ArticleByName(context.Background(), ref)
	if err != nil {
		t.Fatalf("ArticleByName(%q) err = %v, want nil", ref, err)
	}
	return a.ID
}

func TestBreadcrumbsReturnsRootFirst(t *testing.T) {
	d := newTestDB(t)

	got, err := d.Breadcrumbs(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("Breadcrumbs() err = %v, want nil", err)
	}
	want := []string{"probe:parent", "probe:full"}
	if len(got) != len(want) {
		t.Fatalf("len(Breadcrumbs(probe:full)) = %d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].FullName() != name {
			t.Errorf("Breadcrumbs(probe:full)[%d] = %q, want %q", i, got[i].FullName(), name)
		}
	}
}

func TestBreadcrumbsOfPageWithoutParent(t *testing.T) {
	d := newTestDB(t)

	got, err := d.Breadcrumbs(context.Background(), articleID(t, d, "probe:parent"))
	if err != nil {
		t.Fatalf("Breadcrumbs() err = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Breadcrumbs(probe:parent)) = %d, want 1", len(got))
	}
	if got[0].FullName() != "probe:parent" {
		t.Errorf("Breadcrumbs(probe:parent)[0] = %q, want %q", got[0].FullName(), "probe:parent")
	}
}

func TestArticleTagCategoriesOrdersPriorityBeforeUnset(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTagCategories(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("ArticleTagCategories() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ArticleTagCategories(probe:full)) = %d, want 2", len(got))
	}
	if got[0].Name != "lang" {
		t.Errorf("ArticleTagCategories(probe:full)[0].Name = %q, want %q", got[0].Name, "lang")
	}
	if got[0].Priority == nil || *got[0].Priority != 1 {
		t.Errorf("ArticleTagCategories(probe:full)[0].Priority = %v, want 1", got[0].Priority)
	}
	if got[1].Priority != nil {
		t.Errorf("ArticleTagCategories(probe:full)[1].Priority = %v, want nil", got[1].Priority)
	}
}

func TestArticleTagCategoriesKeepsTagOrder(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTagCategories(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("ArticleTagCategories() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ArticleTagCategories(probe:full)) = %d, want 2", len(got))
	}
	want := []string{"lang:en", "zeta", "alpha"}
	var full []string
	for _, category := range got {
		for _, tag := range category.Tags {
			full = append(full, tag.FullName)
		}
	}
	if len(full) != len(want) {
		t.Fatalf("len(tags of probe:full) = %d, want %d", len(full), len(want))
	}
	for i := range want {
		if full[i] != want[i] {
			t.Errorf("tags of probe:full [%d] = %q, want %q", i, full[i], want[i])
		}
	}
}

func TestArticleTagCategoriesOfUntaggedPage(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTagCategories(context.Background(), articleID(t, d, "probe:bare"))
	if err != nil {
		t.Fatalf("ArticleTagCategories() err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len(ArticleTagCategories(probe:bare)) = %d, want 0", len(got))
	}
}

func TestLatestRevNumber(t *testing.T) {
	d := newTestDB(t)

	got, err := d.LatestRevNumber(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("LatestRevNumber() err = %v, want nil", err)
	}
	if got != 0 {
		t.Errorf("LatestRevNumber(probe:full) = %d, want 0", got)
	}
}

func TestCategoryIndexedOfUnknownCategory(t *testing.T) {
	d := newTestDB(t)

	got, err := d.CategoryIndexed(context.Background(), "no-such-category")
	if err != nil {
		t.Fatalf("CategoryIndexed() err = %v, want nil", err)
	}
	if !got {
		t.Errorf("CategoryIndexed(no-such-category) = false, want true")
	}
}

func TestUnreadNotificationsCountsOnlyUnviewed(t *testing.T) {
	d := newTestDB(t)

	got, err := d.UnreadNotifications(context.Background(), userID(t, d, "probe-author"))
	if err != nil {
		t.Fatalf("UnreadNotifications() err = %v, want nil", err)
	}
	if got != 1 {
		t.Errorf("UnreadNotifications(probe-author) = %d, want 1", got)
	}
}

func userID(t *testing.T, d *DB, name string) int64 {
	t.Helper()
	u, err := d.UserByName(context.Background(), name)
	if err != nil {
		t.Fatalf("UserByName(%q) err = %v, want nil", name, err)
	}
	return u.ID
}

func TestArticleTagNamesRepeatsPrefixedTags(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTagNames(context.Background(), articleID(t, d, "probe:full"))
	if err != nil {
		t.Fatalf("ArticleTagNames() err = %v, want nil", err)
	}
	want := []string{"zeta", "alpha", "lang:en", "en"}
	if len(got) != len(want) {
		t.Fatalf("len(ArticleTagNames(probe:full)) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ArticleTagNames(probe:full)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestArticleTagNamesOfUntaggedPage(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTagNames(context.Background(), articleID(t, d, "probe:bare"))
	if err != nil {
		t.Fatalf("ArticleTagNames() err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len(ArticleTagNames(probe:bare)) = %d, want 0", len(got))
	}
}

func TestCategoryExists(t *testing.T) {
	d := newTestDB(t)

	got, err := d.CategoryExists(context.Background(), "probestars")
	if err != nil {
		t.Fatalf("CategoryExists() err = %v, want nil", err)
	}
	if !got {
		t.Errorf("CategoryExists(probestars) = false, want true")
	}
}

func TestCategoryExistsOfUnknownCategory(t *testing.T) {
	d := newTestDB(t)

	got, err := d.CategoryExists(context.Background(), "no-such-category")
	if err != nil {
		t.Fatalf("CategoryExists() err = %v, want nil", err)
	}
	if got {
		t.Errorf("CategoryExists(no-such-category) = true, want false")
	}
}
