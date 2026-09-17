package db

import (
	"context"
	"errors"
	"testing"
)

func scratchForumCategory(t *testing.T, d *DB, siteID int64) (sectionID, categoryID int64) {
	t.Helper()
	ctx := context.Background()
	if err := d.pool.QueryRow(ctx, `
INSERT INTO web_forumsection (name, description, "order", is_hidden, is_hidden_for_users, site_id)
VALUES ('Probe Scope Section', '', 0, false, false, $1)
RETURNING id`, siteID).Scan(&sectionID); err != nil {
		t.Fatalf("insert scratch section err = %v, want nil", err)
	}
	if err := d.pool.QueryRow(ctx, `
INSERT INTO web_forumcategory (name, description, "order", is_for_comments, section_id)
VALUES ('Probe Scope Category', '', 0, false, $1)
RETURNING id`, sectionID).Scan(&categoryID); err != nil {
		t.Fatalf("insert scratch category err = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := d.pool.Exec(ctx, `DELETE FROM web_forumcategory WHERE id = $1`, categoryID); err != nil {
			t.Errorf("clean up scratch category err = %v, want nil", err)
		}
		if _, err := d.pool.Exec(ctx, `DELETE FROM web_forumsection WHERE id = $1`, sectionID); err != nil {
			t.Errorf("clean up scratch section err = %v, want nil", err)
		}
	})
	return sectionID, categoryID
}

func TestForumLookupsFindTheirOwnSite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := seedSiteID(t, d)
	section, category := scratchForumCategory(t, d, site)
	thread := scratchThread(t, d)
	post := scratchPost(t, d, thread, "probe")

	if _, err := d.ForumSection(ctx, site, section); err != nil {
		t.Errorf("ForumSection(%d, %d) err = %v, want nil", site, section, err)
	}
	if _, err := d.ForumCategory(ctx, site, category); err != nil {
		t.Errorf("ForumCategory(%d, %d) err = %v, want nil", site, category, err)
	}
	if _, err := d.ForumThread(ctx, site, thread); err != nil {
		t.Errorf("ForumThread(%d, %d) err = %v, want nil", site, thread, err)
	}
	if _, err := d.ForumPost(ctx, site, post); err != nil {
		t.Errorf("ForumPost(%d, %d) err = %v, want nil", site, post, err)
	}
}

func TestForumLookupsSkipAnotherSite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	other := scratchSite(t, d)
	section, category := scratchForumCategory(t, d, seedSiteID(t, d))
	thread := scratchThread(t, d)
	post := scratchPost(t, d, thread, "probe")

	if _, err := d.ForumSection(ctx, other, section); !errors.Is(err, ErrNotFound) {
		t.Errorf("ForumSection(%d, %d) err = %v, want ErrNotFound", other, section, err)
	}
	if _, err := d.ForumCategory(ctx, other, category); !errors.Is(err, ErrNotFound) {
		t.Errorf("ForumCategory(%d, %d) err = %v, want ErrNotFound", other, category, err)
	}
	if _, err := d.ForumThread(ctx, other, thread); !errors.Is(err, ErrNotFound) {
		t.Errorf("ForumThread(%d, %d) err = %v, want ErrNotFound", other, thread, err)
	}
	if _, err := d.ForumPost(ctx, other, post); !errors.Is(err, ErrNotFound) {
		t.Errorf("ForumPost(%d, %d) err = %v, want ErrNotFound", other, post, err)
	}
}
