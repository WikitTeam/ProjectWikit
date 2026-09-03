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

func linkSet(t *testing.T, d *DB, from string) map[string]bool {
	t.Helper()
	rows, err := d.pool.Query(context.Background(), `
SELECT link_type, link_to FROM web_externallink WHERE link_from = $1`, from)
	if err != nil {
		t.Fatalf("read links err = %v, want nil", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var kind, to string
		if err := rows.Scan(&kind, &to); err != nil {
			t.Fatalf("scan link err = %v, want nil", err)
		}
		out[kind+" "+to] = true
	}
	return out
}

func scratchLinkOwner(t *testing.T, d *DB) string {
	t.Helper()
	from := "probe-links-" + time.Now().Format("20060102150405.000000")
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(),
			`DELETE FROM web_externallink WHERE link_from = $1`, from); err != nil {
			t.Errorf("clean up links err = %v, want nil", err)
		}
	})
	return from
}

func TestReplaceArticleLinksWritesBothKinds(t *testing.T) {
	d := writeTestDB(t)
	from := scratchLinkOwner(t, d)

	err := d.ReplaceArticleLinks(context.Background(), from, []ArticleLink{
		{To: "component:box", Kind: LinkInclude},
		{To: "scp-173", Kind: LinkPlain},
	})
	if err != nil {
		t.Fatalf("ReplaceArticleLinks() err = %v, want nil", err)
	}

	got := linkSet(t, d, from)
	for _, want := range []string{"include component:box", "link scp-173"} {
		if !got[want] {
			t.Errorf("links of %q missing %q, got %v", from, want, got)
		}
	}
	if len(got) != 2 {
		t.Errorf("len(links) = %d, want 2", len(got))
	}
}

func TestReplaceArticleLinksDropsWhatIsGone(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	from := scratchLinkOwner(t, d)

	if err := d.ReplaceArticleLinks(ctx, from, []ArticleLink{{To: "old", Kind: LinkPlain}}); err != nil {
		t.Fatalf("ReplaceArticleLinks(first) err = %v, want nil", err)
	}
	if err := d.ReplaceArticleLinks(ctx, from, []ArticleLink{{To: "new", Kind: LinkPlain}}); err != nil {
		t.Fatalf("ReplaceArticleLinks(second) err = %v, want nil", err)
	}

	got := linkSet(t, d, from)
	if got["link old"] {
		t.Errorf("links of %q still carry the dropped one, got %v", from, got)
	}
	if !got["link new"] {
		t.Errorf("links of %q missing %q, got %v", from, "link new", got)
	}
}

func TestReplaceArticleLinksCollapsesRepeats(t *testing.T) {
	d := writeTestDB(t)
	from := scratchLinkOwner(t, d)

	err := d.ReplaceArticleLinks(context.Background(), from, []ArticleLink{
		{To: "component:box", Kind: LinkInclude},
		{To: "component:box", Kind: LinkInclude},
		{To: "component:box", Kind: LinkPlain},
	})
	if err != nil {
		t.Fatalf("ReplaceArticleLinks() err = %v, want nil", err)
	}
	if got := linkSet(t, d, from); len(got) != 2 {
		t.Errorf("len(links) = %d, want 2, got %v", len(got), got)
	}
}

func TestReplaceArticleLinksEmptiesTheSet(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	from := scratchLinkOwner(t, d)

	if err := d.ReplaceArticleLinks(ctx, from, []ArticleLink{{To: "old", Kind: LinkPlain}}); err != nil {
		t.Fatalf("ReplaceArticleLinks(first) err = %v, want nil", err)
	}
	if err := d.ReplaceArticleLinks(ctx, from, nil); err != nil {
		t.Fatalf("ReplaceArticleLinks(none) err = %v, want nil", err)
	}
	if got := linkSet(t, d, from); len(got) != 0 {
		t.Errorf("len(links) = %d, want 0, got %v", len(got), got)
	}
}

func dropArticle(t *testing.T, d *DB, id int64) {
	t.Helper()
	t.Cleanup(func() {
		clean := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_articlelogentry WHERE article_id = $1`,
			`DELETE FROM web_articleversion WHERE article_id = $1`,
			`DELETE FROM web_article_authors WHERE article_id = $1`,
			`DELETE FROM web_article WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(clean, sql, id); err != nil {
				t.Errorf("clean up article err = %v, want nil", err)
			}
		}
	})
}

func TestCreateArticleCreditsTheAuthor(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()

	var author int64
	if err := d.pool.QueryRow(ctx, `SELECT id FROM web_user ORDER BY id LIMIT 1`).Scan(&author); err != nil {
		t.Fatalf("read a user err = %v, want nil", err)
	}
	name := "probe-new-" + time.Now().Format("20060102150405.000000")

	id, err := d.CreateArticle(ctx, "_default", name, "Probe", &author, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateArticle() err = %v, want nil", err)
	}
	dropArticle(t, d, id)

	var authors int
	if err := d.pool.QueryRow(ctx,
		`SELECT count(*) FROM web_article_authors WHERE article_id = $1 AND user_id = $2`,
		id, author).Scan(&authors); err != nil {
		t.Fatalf("read authors err = %v, want nil", err)
	}
	if authors != 1 {
		t.Errorf("count(authors) = %d, want 1", authors)
	}
}

func TestCreateArticleWithoutAnAuthor(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	name := "probe-new-" + time.Now().Format("20060102150405.000000")

	id, err := d.CreateArticle(ctx, "_default", name, "Probe", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateArticle() err = %v, want nil", err)
	}
	dropArticle(t, d, id)

	article, err := d.ArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("ArticleByID() err = %v, want nil", err)
	}
	if article.Title != "Probe" {
		t.Errorf("ArticleByID().Title = %q, want %q", article.Title, "Probe")
	}
	if article.Locked {
		t.Error("ArticleByID().Locked = true, want false")
	}
}

func TestCreateArticleNamesTheMediaDirectoryApart(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	stamp := time.Now().Format("20060102150405.000000")

	first, err := d.CreateArticle(ctx, "_default", "probe-media-a-"+stamp, "A", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateArticle(a) err = %v, want nil", err)
	}
	dropArticle(t, d, first)
	second, err := d.CreateArticle(ctx, "_default", "probe-media-b-"+stamp, "B", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateArticle(b) err = %v, want nil", err)
	}
	dropArticle(t, d, second)

	var a, b string
	if err := d.pool.QueryRow(ctx, `SELECT media_name FROM web_article WHERE id = $1`, first).Scan(&a); err != nil {
		t.Fatalf("read media_name err = %v, want nil", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT media_name FROM web_article WHERE id = $1`, second).Scan(&b); err != nil {
		t.Fatalf("read media_name err = %v, want nil", err)
	}
	if a == b {
		t.Errorf("media_name of two articles = %q for both, want them apart", a)
	}
	if len(a) != 36 {
		t.Errorf("len(media_name) = %d, want 36", len(a))
	}
}

func TestSetArticleParentRecordsTheMove(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	at := time.Now().UTC().Truncate(time.Second)
	stamp := time.Now().Format("20060102150405.000000")

	parent, err := d.CreateArticle(ctx, "_default", "probe-parent-"+stamp, "Parent", nil, at)
	if err != nil {
		t.Fatalf("CreateArticle(parent) err = %v, want nil", err)
	}
	dropArticle(t, d, parent)
	child, err := d.CreateArticle(ctx, "_default", "probe-child-"+stamp, "Child", nil, at)
	if err != nil {
		t.Fatalf("CreateArticle(child) err = %v, want nil", err)
	}
	dropArticle(t, d, child)

	rev, err := d.SetArticleParent(ctx, child, &parent, nil, `{"parent":"probe-parent"}`, at)
	if err != nil {
		t.Fatalf("SetArticleParent() err = %v, want nil", err)
	}
	if rev != 0 {
		t.Errorf("SetArticleParent().RevNumber = %d, want 0", rev)
	}

	article, err := d.ArticleByID(ctx, child)
	if err != nil {
		t.Fatalf("ArticleByID() err = %v, want nil", err)
	}
	if article.ParentID == nil || *article.ParentID != parent {
		t.Errorf("ArticleByID().ParentID = %v, want %d", article.ParentID, parent)
	}

	var kind string
	if err := d.pool.QueryRow(ctx,
		`SELECT type FROM web_articlelogentry WHERE article_id = $1 AND rev_number = 0`,
		child).Scan(&kind); err != nil {
		t.Fatalf("read revision err = %v, want nil", err)
	}
	if kind != LogParent {
		t.Errorf("revision type = %q, want %q", kind, LogParent)
	}
}

func TestSubscribeToArticleOnlyOnce(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()

	var user int64
	if err := d.pool.QueryRow(ctx, `SELECT id FROM web_user ORDER BY id LIMIT 1`).Scan(&user); err != nil {
		t.Fatalf("read a user err = %v, want nil", err)
	}
	id, err := d.CreateArticle(ctx, "_default",
		"probe-sub-"+time.Now().Format("20060102150405.000000"), "Probe", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateArticle() err = %v, want nil", err)
	}
	dropArticle(t, d, id)
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(),
			`DELETE FROM web_usernotificationsubscription WHERE article_id = $1`, id); err != nil {
			t.Errorf("clean up subscription err = %v, want nil", err)
		}
	})

	for i := 0; i < 2; i++ {
		if err := d.SubscribeToArticle(ctx, user, id); err != nil {
			t.Fatalf("SubscribeToArticle(%d) err = %v, want nil", i, err)
		}
	}

	var rows int
	if err := d.pool.QueryRow(ctx,
		`SELECT count(*) FROM web_usernotificationsubscription WHERE article_id = $1 AND subscriber_id = $2`,
		id, user).Scan(&rows); err != nil {
		t.Fatalf("count subscriptions err = %v, want nil", err)
	}
	if rows != 1 {
		t.Errorf("count(subscriptions) = %d, want 1", rows)
	}
}
