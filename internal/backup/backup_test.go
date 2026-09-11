package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/migrate"
)

const envDSN = "PWIKIT_TEST_WRITE_DSN"

func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(envDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the backup test", envDSN)
	}
	return dsn
}

func swapDatabase(t *testing.T, dsn, name string) string {
	t.Helper()
	cut := strings.LastIndex(dsn, "/")
	if cut < 0 {
		t.Fatalf("no database in %q, want a URL style connection string", dsn)
	}
	rest := ""
	if q := strings.Index(dsn[cut:], "?"); q >= 0 {
		rest = dsn[cut+q:]
	}
	return dsn[:cut+1] + name + rest
}

func scratch(t *testing.T) string {
	t.Helper()
	dsn := requireDSN(t)
	name := fmt.Sprintf("pwikit_backup_%d", rand.Uint32())
	admin := swapDatabase(t, dsn, "postgres")

	ctx := context.Background()
	control, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Skipf("cannot reach the maintenance database to make a scratch one: %v", err)
	}
	defer control.Close(ctx)

	if _, err := control.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("CREATE DATABASE err = %v, want nil", err)
	}
	t.Cleanup(func() {
		clean, err := pgx.Connect(context.Background(), admin)
		if err != nil {
			return
		}
		defer clean.Close(context.Background())
		clean.Exec(context.Background(),
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, name)
		clean.Exec(context.Background(), `DROP DATABASE IF EXISTS `+pgx.Identifier{name}.Sanitize())
	})
	return swapDatabase(t, dsn, name)
}

func connect(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Connect() err = %v, want nil", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

func filled(t *testing.T) (dsn string, files string) {
	t.Helper()
	dsn = scratch(t)
	if _, err := migrate.Run(context.Background(), dsn); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Connect() err = %v, want nil", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
INSERT INTO web_site (slug, title, headline, domain, media_domain, home_page,
	footer_license, signup_notice, password_help, email_policy, membership_password,
	membership_password_enabled, language)
VALUES ('probe', 'Probe', 'p', 'probe.test', 'media.probe.test', 'main', '', '', '', 'optional', '', false, 'zh-hans')`)
	if err != nil {
		t.Fatalf("insert site err = %v, want nil", err)
	}
	_, err = conn.Exec(ctx, `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, type, bio, is_forum_active, is_active, can_send_direct_messages,
	pending_email, previous_email, language)
VALUES ('!', false, '', '', '', now(), 'probe-backup', 'normal', '', true, true, true, '', '', '')`)
	if err != nil {
		t.Fatalf("insert user err = %v, want nil", err)
	}

	files = t.TempDir()
	if err := os.MkdirAll(filepath.Join(files, "media", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"one.txt":              "first",
		"media/deep/two.bin":   "\x00\x01\x02binary",
		"media/deep/three.txt": strings.Repeat("x", 5000),
	} {
		if err := os.WriteFile(filepath.Join(files, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dsn, files
}

func create(t *testing.T, dsn, files string) (string, Manifest) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "probe"+Extension)
	result, err := Create(context.Background(), CreateOptions{DSN: dsn, Files: files, Output: out})
	if err != nil {
		t.Fatalf("Create() err = %v, want nil", err)
	}
	return result.Path, result.Manifest
}

func TestCreateThenVerifyThenRestore(t *testing.T) {
	source, files := filled(t)
	name, made := create(t, source, files)

	if made.Files.Count != 3 {
		t.Errorf("Create().Files.Count = %d, want 3", made.Files.Count)
	}
	if made.Tables["web_site"].Rows != 1 {
		t.Errorf("Create().Tables[web_site].Rows = %d, want 1", made.Tables["web_site"].Rows)
	}
	if len(made.Migrations) != len(migrate.Names()) {
		t.Errorf("len(Create().Migrations) = %d, want %d", len(made.Migrations), len(migrate.Names()))
	}

	report, err := Verify(name)
	if err != nil {
		t.Fatalf("Verify() err = %v, want nil", err)
	}
	if !report.OK() {
		t.Fatalf("Verify().Problems = %v, want none", report.Problems)
	}

	target := scratch(t)
	into := filepath.Join(t.TempDir(), "files")
	result, err := Restore(context.Background(), name, RestoreOptions{DSN: target, Files: into})
	if err != nil {
		t.Fatalf("Restore() err = %v, want nil", err)
	}
	if result.Rows != made.TotalRows() {
		t.Errorf("Restore().Rows = %d, want %d", result.Rows, made.TotalRows())
	}
	if result.FilesPut != 3 {
		t.Errorf("Restore().FilesPut = %d, want 3", result.FilesPut)
	}

	conn := connect(t, target)
	var slug string
	if err := conn.QueryRow(context.Background(), `SELECT slug FROM web_site`).Scan(&slug); err != nil {
		t.Fatalf("read the restored site err = %v, want nil", err)
	}
	if slug != "probe" {
		t.Errorf("restored site slug = %q, want %q", slug, "probe")
	}
	body, err := os.ReadFile(filepath.Join(into, "media", "deep", "two.bin"))
	if err != nil {
		t.Fatalf("read the restored file err = %v, want nil", err)
	}
	if string(body) != "\x00\x01\x02binary" {
		t.Errorf("restored file = %q, want the bytes that went in", body)
	}
}

func TestRestoreLeavesTheIdentityCounterPastTheRestoredRows(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)

	target := scratch(t)
	if _, err := Restore(context.Background(), name, RestoreOptions{DSN: target}); err != nil {
		t.Fatalf("Restore() err = %v, want nil", err)
	}
	conn := connect(t, target)
	ctx := context.Background()

	var was, now int64
	if err := conn.QueryRow(ctx, `SELECT max(id) FROM web_user`).Scan(&was); err != nil {
		t.Fatalf("read the restored user err = %v, want nil", err)
	}
	err := conn.QueryRow(ctx, `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, type, bio, is_forum_active, is_active, can_send_direct_messages,
	pending_email, previous_email, language)
VALUES ('!', false, '', '', '', now(), 'probe-next', 'normal', '', true, true, true, '', '', '')
RETURNING id`).Scan(&now)
	if err != nil {
		t.Fatalf("insert after the restore err = %v, want nil", err)
	}
	if now <= was {
		t.Errorf("the next id = %d, want more than %d", now, was)
	}
}

func TestVerifyReportsADamagedTable(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)
	damaged := rewrite(t, name, func(head *tar.Header, body []byte) ([]byte, bool) {
		if head.Name == dataDir+"/web_site"+dataSuffix {
			return append(body, []byte("bogus\n")...), true
		}
		return body, true
	})

	report, err := Verify(damaged)
	if err != nil {
		t.Fatalf("Verify() err = %v, want nil", err)
	}
	if report.OK() {
		t.Fatal("Verify().OK() = true, want false")
	}
	if !mentions(report.Problems, "web_site") {
		t.Errorf("Verify().Problems = %v, want one naming web_site", report.Problems)
	}
}

func TestVerifyReportsADamagedFile(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)
	damaged := rewrite(t, name, func(head *tar.Header, body []byte) ([]byte, bool) {
		if head.Name == filesDir+"/one.txt" {
			return []byte("tampered"), true
		}
		return body, true
	})

	report, err := Verify(damaged)
	if err != nil {
		t.Fatalf("Verify() err = %v, want nil", err)
	}
	if !mentions(report.Problems, "one.txt") {
		t.Errorf("Verify().Problems = %v, want one naming one.txt", report.Problems)
	}
}

func TestVerifyReportsAMissingTable(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)
	damaged := rewrite(t, name, func(head *tar.Header, body []byte) ([]byte, bool) {
		return body, head.Name != dataDir+"/web_site"+dataSuffix
	})

	report, err := Verify(damaged)
	if err != nil {
		t.Fatalf("Verify() err = %v, want nil", err)
	}
	if !mentions(report.Problems, "web_site") {
		t.Errorf("Verify().Problems = %v, want one naming web_site", report.Problems)
	}
}

func TestVerifyReportsAMigrationThisBuildDoesNotCarry(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)
	damaged := rewrite(t, name, func(head *tar.Header, body []byte) ([]byte, bool) {
		if head.Name != ManifestName {
			return body, true
		}
		var m Manifest
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatal(err)
		}
		m.Migrations = append(m.Migrations, "9999_from_the_future.sql")
		raw, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return raw, true
	})

	report, err := Verify(damaged)
	if err != nil {
		t.Fatalf("Verify() err = %v, want nil", err)
	}
	if !mentions(report.Problems, "9999_from_the_future.sql") {
		t.Errorf("Verify().Problems = %v, want one naming the unknown migration", report.Problems)
	}
}

func TestVerifyRejectsAHalfWrittenArchive(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)
	whole, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	cut := filepath.Join(t.TempDir(), "cut"+Extension)
	if err := os.WriteFile(cut, whole[:len(whole)*2/3], 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Verify(cut); err == nil {
		t.Error("Verify(a truncated archive) err = nil, want non-nil")
	}
}

func TestRestoreRefusesADatabaseThatHoldsData(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)

	target, _ := filled(t)
	_, err := Restore(context.Background(), name, RestoreOptions{DSN: target})
	if err == nil {
		t.Fatal("Restore(into a full database) err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "-force") {
		t.Errorf("Restore() err = %v, want it to name -force", err)
	}
}

func TestRestoreReplacesAFullDatabaseWithForce(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)

	target, _ := filled(t)
	conn := connect(t, target)
	ctx := context.Background()
	if _, err := conn.Exec(ctx, `UPDATE web_site SET title = 'stale'`); err != nil {
		t.Fatal(err)
	}
	conn.Close(ctx)

	if _, err := Restore(ctx, name, RestoreOptions{DSN: target, Force: true}); err != nil {
		t.Fatalf("Restore(-force) err = %v, want nil", err)
	}
	after := connect(t, target)
	var title string
	if err := after.QueryRow(ctx, `SELECT title FROM web_site`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Probe" {
		t.Errorf("restored title = %q, want %q", title, "Probe")
	}
}

func TestRestoreRefusesWhileSomethingElseIsConnected(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)

	target := scratch(t)
	holding := connect(t, target)
	if _, err := holding.Exec(context.Background(), `SELECT 1`); err != nil {
		t.Fatal(err)
	}

	_, err := Restore(context.Background(), name, RestoreOptions{DSN: target})
	if err == nil {
		t.Fatal("Restore(with another connection open) err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "stop pwikit") {
		t.Errorf("Restore() err = %v, want it to say what to stop", err)
	}
}

func TestRestoreChangesNothingWhenTheBackupIsDamaged(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)
	damaged := rewrite(t, name, func(head *tar.Header, body []byte) ([]byte, bool) {
		if head.Name == dataDir+"/web_user"+dataSuffix {
			return append(body, []byte("bogus\n")...), true
		}
		return body, true
	})

	target, _ := filled(t)
	before := connect(t, target)
	ctx := context.Background()
	var was string
	if err := before.QueryRow(ctx, `SELECT title FROM web_site`).Scan(&was); err != nil {
		t.Fatal(err)
	}
	before.Close(ctx)

	if _, err := Restore(ctx, damaged, RestoreOptions{DSN: target, Force: true}); err == nil {
		t.Fatal("Restore(a damaged backup) err = nil, want non-nil")
	}
	after := connect(t, target)
	var now string
	if err := after.QueryRow(ctx, `SELECT title FROM web_site`).Scan(&now); err != nil {
		t.Fatalf("the database is unusable after a refused restore: %v", err)
	}
	if now != was {
		t.Errorf("title after a refused restore = %q, want it untouched at %q", now, was)
	}
}

func TestListReportsAnArchiveItCannotRead(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)

	dir := t.TempDir()
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good"+Extension), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "junk"+Extension), []byte("not an archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := List(dir)
	if err != nil {
		t.Fatalf("List() err = %v, want nil", err)
	}
	if len(found) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(found))
	}
	var problems int
	for _, one := range found {
		if one.Problem != "" {
			problems++
		}
	}
	if problems != 1 {
		t.Errorf("List() reported %d unreadable archives, want 1", problems)
	}
}

func rewrite(t *testing.T, name string, change func(*tar.Header, []byte) ([]byte, bool)) string {
	t.Helper()
	in, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	zin, err := gzip.NewReader(in)
	if err != nil {
		t.Fatal(err)
	}
	defer zin.Close()

	out := filepath.Join(t.TempDir(), "changed"+Extension)
	file, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	zout := gzip.NewWriter(file)
	tw := tar.NewWriter(zout)
	tr := tar.NewReader(zin)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		body, keep := change(head, body)
		if !keep {
			continue
		}
		head.Size = int64(len(body))
		if err := tw.WriteHeader(head); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zout.Close(); err != nil {
		t.Fatal(err)
	}
	return out
}

func mentions(problems []string, want string) bool {
	for _, one := range problems {
		if strings.Contains(one, want) {
			return true
		}
	}
	return false
}

func TestEveryTableSaysWhetherItBelongsToASite(t *testing.T) {
	dsn := scratch(t)
	if _, err := migrate.Run(context.Background(), dsn); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Connect() err = %v, want nil", err)
	}
	defer conn.Close(ctx)

	present, err := db.BackupTables(ctx, conn)
	if err != nil {
		t.Fatalf("BackupTables() err = %v, want nil", err)
	}
	ruled := map[string]bool{}
	for _, name := range db.SiteScopeRuled() {
		ruled[name] = true
	}
	for _, name := range present {
		if !ruled[name] {
			t.Errorf("rules[%q] is missing, want a rule saying whether it belongs to a site", name)
		}
		delete(ruled, name)
	}
	for name := range ruled {
		t.Errorf("rules[%q] names a table the schema does not have", name)
	}
}

func TestEveryScopedQueryRuns(t *testing.T) {
	dsn, _ := filled(t)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Connect() err = %v, want nil", err)
	}
	defer conn.Close(ctx)

	present, err := db.BackupTables(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range present {
		for _, keep := range []bool{true, false} {
			query, err := db.SiteExportQuery(ctx, conn, name, keep)
			if err != nil {
				t.Errorf("SiteExportQuery(%q) err = %v, want nil", name, err)
				continue
			}
			if query == "" {
				continue
			}
			if _, err := conn.Exec(ctx, `SELECT count(*) FROM (`+
				strings.ReplaceAll(query, "$1", `'probe'`)+`) q`); err != nil {
				t.Errorf("the rule for %q does not run: %v", name, err)
			}
		}
	}
}

func twoSites(t *testing.T) string {
	t.Helper()
	dsn := scratch(t)
	if _, err := migrate.Run(context.Background(), dsn); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Connect() err = %v, want nil", err)
	}
	defer conn.Close(ctx)

	for _, slug := range []string{"leaving", "staying"} {
		_, err := conn.Exec(ctx, `
INSERT INTO web_site (slug, title, headline, domain, media_domain, home_page,
	footer_license, signup_notice, password_help, email_policy, membership_password,
	membership_password_enabled, language)
VALUES ($1, $1, '', $1 || '.test', 'media.' || $1 || '.test', 'main', '', '', '', 'optional', '', false, 'zh-hans')`, slug)
		if err != nil {
			t.Fatalf("insert site %s err = %v, want nil", slug, err)
		}
		var user int64
		err = conn.QueryRow(ctx, `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, type, bio, is_forum_active, is_active, can_send_direct_messages,
	pending_email, previous_email, language)
VALUES ('secret-hash', false, '', '', '', now(), $1 || '-author', 'normal', '', true, true, true, '', '', '')
RETURNING id`, slug).Scan(&user)
		if err != nil {
			t.Fatalf("insert user for %s err = %v, want nil", slug, err)
		}
		var article int64
		err = conn.QueryRow(ctx, `
INSERT INTO web_article (site_id, category, name, title, locked, created_at, updated_at, media_name)
VALUES ((SELECT id FROM web_site WHERE slug = $1), '_default', $1 || '-page', $1, false, now(), now(), $1)
RETURNING id`, slug).Scan(&article)
		if err != nil {
			t.Fatalf("insert article for %s err = %v, want nil", slug, err)
		}
		if _, err := conn.Exec(ctx,
			`INSERT INTO web_article_authors (article_id, user_id) VALUES ($1, $2)`, article, user); err != nil {
			t.Fatalf("insert authorship for %s err = %v, want nil", slug, err)
		}
	}
	_, err = conn.Exec(ctx, `
INSERT INTO web_externallink (link_from, link_to, link_type, from_site_id, to_site_id)
VALUES ('leaving:page', 'staying:page', 'internal',
	(SELECT id FROM web_site WHERE slug = 'leaving'),
	(SELECT id FROM web_site WHERE slug = 'staying'))`)
	if err != nil {
		t.Fatalf("insert cross site link err = %v, want nil", err)
	}

	_, err = conn.Exec(ctx, `
INSERT INTO web_directmessage (sender_id, recipient_id, body, created_at, is_read)
SELECT a.id, b.id, 'private', now(), false
FROM web_user a, web_user b WHERE a.username = 'leaving-author' AND b.username = 'staying-author'`)
	if err != nil {
		t.Fatalf("insert direct message err = %v, want nil", err)
	}
	return dsn
}

func TestSiteBackupCarriesOneSiteAndLeavesTheOther(t *testing.T) {
	source := twoSites(t)
	out := filepath.Join(t.TempDir(), "one"+Extension)
	result, err := Create(context.Background(), CreateOptions{DSN: source, Output: out, Site: "leaving"})
	if err != nil {
		t.Fatalf("Create(-site) err = %v, want nil", err)
	}
	if result.Manifest.Site != "leaving" {
		t.Errorf("Create().Manifest.Site = %q, want %q", result.Manifest.Site, "leaving")
	}
	if got := result.Manifest.Tables["web_site"].Rows; got != 1 {
		t.Errorf("web_site rows = %d, want 1", got)
	}
	if got := result.Manifest.Tables["web_directmessage"].Rows; got != 0 {
		t.Errorf("web_directmessage rows = %d, want 0", got)
	}

	report, err := Verify(out)
	if err != nil {
		t.Fatalf("Verify() err = %v, want nil", err)
	}
	if !report.OK() {
		t.Fatalf("Verify().Problems = %v, want none", report.Problems)
	}

	target := scratch(t)
	if _, err := Restore(context.Background(), out, RestoreOptions{DSN: target}); err != nil {
		t.Fatalf("Restore() err = %v, want nil", err)
	}
	conn := connect(t, target)
	ctx := context.Background()

	var slugs string
	if err := conn.QueryRow(ctx, `SELECT coalesce(string_agg(slug, ','), '') FROM web_site`).Scan(&slugs); err != nil {
		t.Fatal(err)
	}
	if slugs != "leaving" {
		t.Errorf("restored sites = %q, want %q", slugs, "leaving")
	}
	var names string
	if err := conn.QueryRow(ctx, `SELECT coalesce(string_agg(username, ','), '') FROM web_user ORDER BY 1`).Scan(&names); err != nil {
		t.Fatal(err)
	}
	if names != "leaving-author" {
		t.Errorf("restored users = %q, want only the one the site points at", names)
	}
	var mail int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM web_directmessage`).Scan(&mail); err != nil {
		t.Fatal(err)
	}
	if mail != 0 {
		t.Errorf("restored direct messages = %d, want 0", mail)
	}

	var from, to *int64
	err = conn.QueryRow(ctx, `SELECT from_site_id, to_site_id FROM web_externallink`).Scan(&from, &to)
	if err != nil {
		t.Fatalf("read the restored link err = %v, want nil", err)
	}
	if from == nil {
		t.Error("the restored link has no site it came from, want one")
	}
	if to != nil {
		t.Errorf("the restored link still names site %d as its target, want none", *to)
	}
}

func TestSiteBackupBlanksThePasswordUnlessAsked(t *testing.T) {
	source := twoSites(t)
	ctx := context.Background()

	for _, keep := range []bool{false, true} {
		out := filepath.Join(t.TempDir(), "one"+Extension)
		if _, err := Create(ctx, CreateOptions{DSN: source, Output: out, Site: "leaving", KeepPasswords: keep}); err != nil {
			t.Fatalf("Create(keep=%t) err = %v, want nil", keep, err)
		}
		target := scratch(t)
		if _, err := Restore(ctx, out, RestoreOptions{DSN: target}); err != nil {
			t.Fatalf("Restore(keep=%t) err = %v, want nil", keep, err)
		}
		conn, err := pgx.Connect(ctx, target)
		if err != nil {
			t.Fatal(err)
		}
		var stored string
		if err := conn.QueryRow(ctx, `SELECT password FROM web_user WHERE username = 'leaving-author'`).Scan(&stored); err != nil {
			conn.Close(ctx)
			t.Fatal(err)
		}
		conn.Close(ctx)

		want := "!"
		if keep {
			want = "secret-hash"
		}
		if stored != want {
			t.Errorf("password with KeepPasswords=%t = %q, want %q", keep, stored, want)
		}
	}
}

func TestSiteBackupLeavesTheOldOperatorsRightsBehind(t *testing.T) {
	source := twoSites(t)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(ctx,
		`UPDATE web_user SET is_superuser = true, api_key = 'operator-key' WHERE username = 'leaving-author'`)
	conn.Close(ctx)
	if err != nil {
		t.Fatalf("make the author a superuser err = %v, want nil", err)
	}

	out := filepath.Join(t.TempDir(), "one"+Extension)
	if _, err := Create(ctx, CreateOptions{DSN: source, Output: out, Site: "leaving", KeepPasswords: true}); err != nil {
		t.Fatalf("Create(-site) err = %v, want nil", err)
	}
	target := scratch(t)
	if _, err := Restore(ctx, out, RestoreOptions{DSN: target}); err != nil {
		t.Fatalf("Restore() err = %v, want nil", err)
	}
	after := connect(t, target)

	var super bool
	var key *string
	err = after.QueryRow(ctx,
		`SELECT is_superuser, api_key FROM web_user WHERE username = 'leaving-author'`).Scan(&super, &key)
	if err != nil {
		t.Fatal(err)
	}
	if super {
		t.Error("is_superuser after a site backup = true, want false")
	}
	if key != nil {
		t.Errorf("api_key after a site backup = %q, want none", *key)
	}
}

func TestCheckServerLetsAServerNewEnoughThrough(t *testing.T) {
	dsn := requireDSN(t)
	found, err := CheckServer(context.Background(), dsn)
	if err != nil {
		t.Fatalf("CheckServer() err = %v, want nil", err)
	}
	if found < MinimumPGVersion {
		t.Errorf("CheckServer() = %d, want at least %d", found, MinimumPGVersion)
	}
}

func TestTooOldSaysWhatToRun(t *testing.T) {
	err := TooOld(120000)
	for _, want := range []string{"12.0", "14.0", "pwikit backup create", "pwikit backup restore"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Contains(TooOld(120000), %q) = false, want true", want)
		}
	}
}

func TestDescribeReadsBackAsAVersion(t *testing.T) {
	for _, c := range []struct {
		num  int
		want string
	}{{140023, "14.23"}, {170000, "17.0"}, {0, "unknown"}} {
		if got := Describe(c.num); got != c.want {
			t.Errorf("Describe(%d) = %q, want %q", c.num, got, c.want)
		}
	}
}

func TestReadyOnABlankDatabase(t *testing.T) {
	target := scratch(t)
	holdsData, err := Ready(context.Background(), target, false)
	if err != nil {
		t.Fatalf("Ready(blank) err = %v, want nil", err)
	}
	if holdsData {
		t.Error("Ready(blank) holdsData = true, want false")
	}
}

func TestReadyOnADatabaseOnlyMigrated(t *testing.T) {
	target := scratch(t)
	ctx := context.Background()
	if _, err := migrate.Run(ctx, target); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	holdsData, err := Ready(ctx, target, false)
	if err != nil {
		t.Fatalf("Ready(migrated) err = %v, want nil", err)
	}
	if holdsData {
		t.Error("Ready(migrated) holdsData = true, want false")
	}
}

func TestSeededNamesEveryTableTheMigrationsFill(t *testing.T) {
	target := scratch(t)
	ctx := context.Background()
	if _, err := migrate.Run(ctx, target); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	conn := connect(t, target)
	tables, err := db.BackupTables(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	full, err := db.NonEmptyTables(ctx, conn, tables)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range full {
		if !seeded[name] {
			t.Errorf("seeded[%q] = false, want true", name)
		}
	}
}

func TestReadyOnADatabaseThatHoldsData(t *testing.T) {
	target, _ := filled(t)
	holdsData, err := Ready(context.Background(), target, true)
	if err != nil {
		t.Fatalf("Ready(full, force) err = %v, want nil", err)
	}
	if !holdsData {
		t.Error("Ready(full, force) holdsData = false, want true")
	}
}

func TestRestoreIntoAMigratedDatabaseWithoutForce(t *testing.T) {
	source, files := filled(t)
	name, _ := create(t, source, files)

	target := scratch(t)
	ctx := context.Background()
	if _, err := migrate.Run(ctx, target); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if _, err := Restore(ctx, name, RestoreOptions{DSN: target}); err != nil {
		t.Fatalf("Restore(into a migrated database) err = %v, want nil", err)
	}
	conn := connect(t, target)
	var slug string
	if err := conn.QueryRow(ctx, `SELECT slug FROM web_site`).Scan(&slug); err != nil {
		t.Fatalf("read the restored site err = %v, want nil", err)
	}
	if slug != "probe" {
		t.Errorf("restored site slug = %q, want %q", slug, "probe")
	}
}
