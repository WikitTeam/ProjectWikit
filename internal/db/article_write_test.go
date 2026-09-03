package db

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func writeTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	conn, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	t.Cleanup(conn.Close)
	return conn
}

func scratchArticle(t *testing.T, d *DB) int64 {
	t.Helper()
	ctx := context.Background()
	name := "probe-write-" + time.Now().Format("20060102150405.000000")
	var id int64
	err := d.pool.QueryRow(ctx, `
INSERT INTO web_article (category, name, title, locked, created_at, updated_at, media_name)
VALUES ('_default', $1, 'Probe', false, now(), now(), $2)
RETURNING id`, name, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert scratch article err = %v, want nil", err)
	}
	t.Cleanup(func() {
		clean := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_articlelogentry WHERE article_id = $1`,
			`DELETE FROM web_articleversion WHERE article_id = $1`,
			`DELETE FROM web_article WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(clean, sql, id); err != nil {
				t.Errorf("clean up scratch article err = %v, want nil", err)
			}
		}
	})
	return id
}

func TestCreateArticleVersionNumbersFromZero(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchArticle(t, d)
	at := time.Now().UTC().Truncate(time.Second)

	first, err := d.CreateArticleVersion(ctx, VersionWrite{
		ArticleID: id, Source: "one", Kind: LogNew, Title: "Probe", At: at,
	})
	if err != nil {
		t.Fatalf("CreateArticleVersion(new) err = %v, want nil", err)
	}
	if first.RevNumber != 0 {
		t.Errorf("CreateArticleVersion(new).RevNumber = %d, want 0", first.RevNumber)
	}

	second, err := d.CreateArticleVersion(ctx, VersionWrite{
		ArticleID: id, Source: "two", Kind: LogSource, Comment: "fixed a typo", At: at,
	})
	if err != nil {
		t.Fatalf("CreateArticleVersion(source) err = %v, want nil", err)
	}
	if second.RevNumber != 1 {
		t.Errorf("CreateArticleVersion(source).RevNumber = %d, want 1", second.RevNumber)
	}
	if second.VersionID == first.VersionID {
		t.Errorf("CreateArticleVersion(source).VersionID = %d, want a new one", second.VersionID)
	}

	source, err := d.LatestSource(ctx, id)
	if err != nil {
		t.Fatalf("LatestSource() err = %v, want nil", err)
	}
	if source != "two" {
		t.Errorf("LatestSource() = %q, want %q", source, "two")
	}
}

func TestCreateArticleVersionMetaNamesTheVersion(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchArticle(t, d)
	at := time.Now().UTC().Truncate(time.Second)

	rev, err := d.CreateArticleVersion(ctx, VersionWrite{
		ArticleID: id, Source: "one", Kind: LogNew, Title: "Probe", At: at,
	})
	if err != nil {
		t.Fatalf("CreateArticleVersion(new) err = %v, want nil", err)
	}

	var kind, comment string
	var raw []byte
	err = d.pool.QueryRow(ctx, `
SELECT type, comment, meta FROM web_articlelogentry WHERE article_id = $1 AND rev_number = 0`,
		id).Scan(&kind, &comment, &raw)
	if err != nil {
		t.Fatalf("read revision err = %v, want nil", err)
	}
	if kind != LogNew {
		t.Errorf("revision type = %q, want %q", kind, LogNew)
	}

	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("decode meta err = %v, want nil", err)
	}
	if got, want := meta["version_id"], float64(rev.VersionID); got != want {
		t.Errorf("meta version_id = %v, want %v", got, want)
	}
	if got, want := meta["title"], "Probe"; got != want {
		t.Errorf("meta title = %v, want %q", got, want)
	}
}

func TestCreateArticleVersionKeepsTheTitleOffAnEdit(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchArticle(t, d)
	at := time.Now().UTC().Truncate(time.Second)

	if _, err := d.CreateArticleVersion(ctx, VersionWrite{
		ArticleID: id, Source: "one", Kind: LogNew, Title: "Probe", At: at,
	}); err != nil {
		t.Fatalf("CreateArticleVersion(new) err = %v, want nil", err)
	}
	if _, err := d.CreateArticleVersion(ctx, VersionWrite{
		ArticleID: id, Source: "two", Kind: LogSource, Title: "Probe", At: at,
	}); err != nil {
		t.Fatalf("CreateArticleVersion(source) err = %v, want nil", err)
	}

	var raw []byte
	err := d.pool.QueryRow(ctx, `
SELECT meta FROM web_articlelogentry WHERE article_id = $1 AND rev_number = 1`, id).Scan(&raw)
	if err != nil {
		t.Fatalf("read revision err = %v, want nil", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("decode meta err = %v, want nil", err)
	}
	if _, ok := meta["title"]; ok {
		t.Errorf("meta of an edit carries title = %v, want it absent", meta["title"])
	}
}

func TestCreateArticleVersionTouchesTheArticle(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchArticle(t, d)
	at := time.Date(2030, 4, 5, 6, 7, 8, 0, time.UTC)

	if _, err := d.CreateArticleVersion(ctx, VersionWrite{
		ArticleID: id, Source: "one", Kind: LogNew, At: at,
	}); err != nil {
		t.Fatalf("CreateArticleVersion(new) err = %v, want nil", err)
	}

	var updated time.Time
	if err := d.pool.QueryRow(ctx, `SELECT updated_at FROM web_article WHERE id = $1`, id).Scan(&updated); err != nil {
		t.Fatalf("read article err = %v, want nil", err)
	}
	if !updated.Equal(at) {
		t.Errorf("article updated_at = %v, want %v", updated, at)
	}
}
