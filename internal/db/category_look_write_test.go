package db

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func scratchCategoryTheme(t *testing.T, d *DB, siteID int64) int64 {
	t.Helper()
	slug := "probe-theme-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	id, err := d.SaveTheme(context.Background(), siteID, ThemeRow{Name: slug, Slug: slug, Mode: ThemeInline, CSS: "body{}"})
	if err != nil {
		t.Fatalf("SaveTheme() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_theme WHERE id = $1`, id); err != nil {
			t.Errorf("clean up theme err = %v, want nil", err)
		}
	})
	return id
}

func scratchCategory(t *testing.T, d *DB, siteID int64, row CategoryRow) CategoryRow {
	t.Helper()
	ctx := context.Background()
	row.Name = "probe-look-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	row.Settings = SiteSettings{RatingMode: "follow", CreateTags: "follow"}
	if err := d.SaveCategory(ctx, siteID, row); err != nil {
		t.Fatalf("SaveCategory() err = %v, want nil", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT id FROM web_category WHERE site_id = $1 AND name = $2`, siteID, row.Name).Scan(&row.ID); err != nil {
		t.Fatalf("find saved category err = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := d.pool.Exec(ctx, `DELETE FROM web_settings WHERE category_id = $1`, row.ID); err != nil {
			t.Errorf("clean up category settings err = %v, want nil", err)
		}
		if _, err := d.pool.Exec(ctx, `DELETE FROM web_category WHERE id = $1`, row.ID); err != nil {
			t.Errorf("clean up category err = %v, want nil", err)
		}
	})
	return row
}

func TestCategoryLookReadsWhatWasSaved(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := seedSiteID(t, d)
	theme := scratchCategoryTheme(t, d, site)
	row := scratchCategory(t, d, site, CategoryRow{IsIndexed: true, ThemeID: &theme, NavTop: "nav:top-probe", NavSide: "nav:side-probe"})

	look, err := d.CategoryLook(ctx, site, row.Name)
	if err != nil {
		t.Fatalf("CategoryLook(%q) err = %v, want nil", row.Name, err)
	}
	if look.ThemeID == nil || *look.ThemeID != theme {
		t.Errorf("CategoryLook(%q).ThemeID = %v, want %d", row.Name, look.ThemeID, theme)
	}
	if look.NavTop != "nav:top-probe" {
		t.Errorf("CategoryLook(%q).NavTop = %q, want %q", row.Name, look.NavTop, "nav:top-probe")
	}
	if look.NavSide != "nav:side-probe" {
		t.Errorf("CategoryLook(%q).NavSide = %q, want %q", row.Name, look.NavSide, "nav:side-probe")
	}

	admin, err := d.AdminCategory(ctx, site, row.ID)
	if err != nil {
		t.Fatalf("AdminCategory(%d) err = %v, want nil", row.ID, err)
	}
	if admin.NavSide != "nav:side-probe" {
		t.Errorf("AdminCategory(%d).NavSide = %q, want %q", row.ID, admin.NavSide, "nav:side-probe")
	}
}

func TestCategoryLookOfAnUnsavedCategoryIsEmpty(t *testing.T) {
	d := writeTestDB(t)
	look, err := d.CategoryLook(context.Background(), seedSiteID(t, d), "probe-never-saved")
	if err != nil {
		t.Fatalf("CategoryLook() err = %v, want nil", err)
	}
	if look != (CategoryLook{}) {
		t.Errorf("CategoryLook() = %+v, want zero", look)
	}
}

func TestDeleteThemeDetachesCategories(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := seedSiteID(t, d)
	theme := scratchCategoryTheme(t, d, site)
	row := scratchCategory(t, d, site, CategoryRow{IsIndexed: true, ThemeID: &theme})

	if err := d.DeleteTheme(ctx, site, theme); err != nil {
		t.Fatalf("DeleteTheme(%d) err = %v, want nil", theme, err)
	}
	look, err := d.CategoryLook(ctx, site, row.Name)
	if err != nil {
		t.Fatalf("CategoryLook(%q) err = %v, want nil", row.Name, err)
	}
	if look.ThemeID != nil {
		t.Errorf("CategoryLook(%q).ThemeID = %d, want nil", row.Name, *look.ThemeID)
	}
}
