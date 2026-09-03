package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func tagNamesOf(t *testing.T, d *DB, articleID int64) []string {
	t.Helper()
	rows, err := d.pool.Query(context.Background(), `
SELECT c.slug, t.name
FROM web_article_tags at
JOIN web_tag t ON t.id = at.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE at.article_id = $1
ORDER BY c.slug, t.name`, articleID)
	if err != nil {
		t.Fatalf("read tags err = %v, want nil", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var category, name string
		if err := rows.Scan(&category, &name); err != nil {
			t.Fatalf("scan tag err = %v, want nil", err)
		}
		out = append(out, tagFullName(category, name))
	}
	return out
}

func scratchTagged(t *testing.T, d *DB) int64 {
	t.Helper()
	id, err := d.CreateArticle(context.Background(), "_default",
		"probe-tags-"+time.Now().Format("20060102150405.000000"), "Probe", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateArticle() err = %v, want nil", err)
	}
	dropArticle(t, d, id)
	t.Cleanup(func() {
		clean := context.Background()
		if _, err := d.pool.Exec(clean, `DELETE FROM web_article_tags WHERE article_id = $1`, id); err != nil {
			t.Errorf("clean up tags err = %v, want nil", err)
		}
		if _, err := d.pool.Exec(clean, qDropOrphanTags); err != nil {
			t.Errorf("sweep tags err = %v, want nil", err)
		}
		if _, err := d.pool.Exec(clean, qDropOrphanTagCategories); err != nil {
			t.Errorf("sweep tag categories err = %v, want nil", err)
		}
	})
	return id
}

func TestSetArticleTagsCreatesWhatItIsAllowedTo(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchTagged(t, d)
	stamp := time.Now().Format("150405.000000")

	_, wrote, err := d.SetArticleTags(ctx, id,
		[]string{"probe" + stamp, "probecat" + stamp + ":one"}, true, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("SetArticleTags() err = %v, want nil", err)
	}
	if !wrote {
		t.Fatal("SetArticleTags() wrote no revision, want one")
	}

	got := tagNamesOf(t, d, id)
	want := []string{"probe" + stamp, "probecat" + stamp + ":one"}
	if len(got) != len(want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tags[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSetArticleTagsSkipsWhatItMayNotCreate(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchTagged(t, d)

	_, wrote, err := d.SetArticleTags(ctx, id,
		[]string{"probe-never-seen-" + time.Now().Format("150405.000000")}, false, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("SetArticleTags() err = %v, want nil", err)
	}
	if wrote {
		t.Error("SetArticleTags() wrote a revision, want none")
	}
	if got := tagNamesOf(t, d, id); len(got) != 0 {
		t.Errorf("tags = %v, want none", got)
	}
}

func TestSetArticleTagsDropsANameWithASpace(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchTagged(t, d)
	stamp := time.Now().Format("150405.000000")

	if _, _, err := d.SetArticleTags(ctx, id,
		[]string{"two words", "probe" + stamp}, true, nil, time.Now().UTC()); err != nil {
		t.Fatalf("SetArticleTags() err = %v, want nil", err)
	}
	got := tagNamesOf(t, d, id)
	if len(got) != 1 || got[0] != "probe"+stamp {
		t.Errorf("tags = %v, want [%q]", got, "probe"+stamp)
	}
}

func TestSetArticleTagsRecordsWhatMoved(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchTagged(t, d)
	stamp := time.Now().Format("150405.000000")
	at := time.Now().UTC()

	if _, _, err := d.SetArticleTags(ctx, id, []string{"probeold" + stamp}, true, nil, at); err != nil {
		t.Fatalf("SetArticleTags(first) err = %v, want nil", err)
	}
	if _, _, err := d.SetArticleTags(ctx, id, []string{"probenew" + stamp}, true, nil, at.Add(time.Second)); err != nil {
		t.Fatalf("SetArticleTags(second) err = %v, want nil", err)
	}

	var raw []byte
	if err := d.pool.QueryRow(ctx,
		`SELECT meta FROM web_articlelogentry WHERE article_id = $1 AND rev_number = 1`, id).Scan(&raw); err != nil {
		t.Fatalf("read revision err = %v, want nil", err)
	}
	var meta map[string][]taggedName
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("decode meta err = %v, want nil", err)
	}
	if len(meta["added_tags"]) != 1 || meta["added_tags"][0].Name != "probenew"+stamp {
		t.Errorf("meta added_tags = %v, want one named %q", meta["added_tags"], "probenew"+stamp)
	}
	if len(meta["removed_tags"]) != 1 || meta["removed_tags"][0].Name != "probeold"+stamp {
		t.Errorf("meta removed_tags = %v, want one named %q", meta["removed_tags"], "probeold"+stamp)
	}
}

func TestSetArticleTagsKeepsQuietWhenNothingMoves(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchTagged(t, d)
	stamp := time.Now().Format("150405.000000")
	at := time.Now().UTC()

	if _, _, err := d.SetArticleTags(ctx, id, []string{"probe" + stamp}, true, nil, at); err != nil {
		t.Fatalf("SetArticleTags(first) err = %v, want nil", err)
	}
	_, wrote, err := d.SetArticleTags(ctx, id, []string{"probe" + stamp}, true, nil, at.Add(time.Second))
	if err != nil {
		t.Fatalf("SetArticleTags(second) err = %v, want nil", err)
	}
	if wrote {
		t.Error("SetArticleTags() wrote a revision for an unchanged set, want none")
	}
}

func TestSetArticleTagsSweepsATagNothingCarries(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchTagged(t, d)
	stamp := time.Now().Format("150405.000000")
	at := time.Now().UTC()

	if _, _, err := d.SetArticleTags(ctx, id, []string{"probegone" + stamp}, true, nil, at); err != nil {
		t.Fatalf("SetArticleTags(first) err = %v, want nil", err)
	}
	if _, _, err := d.SetArticleTags(ctx, id, []string{"probekept" + stamp}, true, nil, at.Add(time.Second)); err != nil {
		t.Fatalf("SetArticleTags(second) err = %v, want nil", err)
	}

	var left int
	if err := d.pool.QueryRow(ctx,
		`SELECT count(*) FROM web_tag WHERE name = $1`, "probegone"+stamp).Scan(&left); err != nil {
		t.Fatalf("count tags err = %v, want nil", err)
	}
	if left != 0 {
		t.Errorf("count(tag nothing carries) = %d, want 0", left)
	}
}
