package db

import (
	"context"
	"testing"
)

func TestUpdateProfileStoresTheLanguage(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	user := scratchUser(t, d, "probe-lang")

	if err := d.UpdateProfile(ctx, user, "Ada", "Lovelace", "", nil, "en"); err != nil {
		t.Fatalf("UpdateProfile() err = %v, want nil", err)
	}
	got, err := d.UserByID(ctx, user)
	if err != nil {
		t.Fatalf("UserByID() err = %v, want nil", err)
	}
	if got.Language != "en" {
		t.Errorf("UserByID().Language = %q, want %q", got.Language, "en")
	}
}

func TestSaveSiteStoresTheLanguage(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()

	slugs, err := d.SiteSlugs(ctx)
	if err != nil || len(slugs) == 0 {
		t.Skipf("SiteSlugs() = %v, %v, skipping without a site", slugs, err)
	}
	before, err := d.SiteBySlug(ctx, slugs[0])
	if err != nil {
		t.Fatalf("SiteBySlug(%q) err = %v, want nil", slugs[0], err)
	}
	settings, err := d.SiteSettings(ctx, before.ID)
	if err != nil {
		t.Fatalf("SiteSettings() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		restore := *before
		if err := d.SaveSite(context.Background(), &restore, settings, true); err != nil {
			t.Errorf("SaveSite(restore) err = %v, want nil", err)
		}
	})

	next := *before
	next.Language = "en"
	if err := d.SaveSite(ctx, &next, settings, true); err != nil {
		t.Fatalf("SaveSite() err = %v, want nil", err)
	}
	after, err := d.SiteBySlug(ctx, slugs[0])
	if err != nil {
		t.Fatalf("SiteBySlug(%q) err = %v, want nil", slugs[0], err)
	}
	if after.Language != "en" {
		t.Errorf("SiteBySlug().Language = %q, want %q", after.Language, "en")
	}
}
