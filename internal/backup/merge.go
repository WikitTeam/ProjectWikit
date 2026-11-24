package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

type ImportOptions struct {
	DSN string
	// An empty database the backup is unpacked into first. Empty asks the
	// server to make one and drop it afterwards.
	Staging string
	// Nil leaves the files behind.
	Storage *storage.Set
	// The sites to take, by their slug in the backup. Empty takes every one.
	Sites []string
	// Only read when one site is taken.
	Slug, Domain, MediaDomain string
	Report                    func(string)
}

type ImportedSite struct {
	From, Slug   string
	Rows         int64
	NewAccounts  int
	SameAccounts int
	Files        int
	// The stored page names that had to change, old to new.
	media map[string]string
}

type ImportResult struct {
	Sites []ImportedSite
}

// The backup is unpacked into a database of its own first, since that is the
// one way to bring an older backup up to this schema.
func Import(ctx context.Context, name string, opts ImportOptions) (ImportResult, error) {
	var out ImportResult
	to, err := pgx.Connect(ctx, opts.DSN)
	if err != nil {
		return out, err
	}
	defer to.Close(ctx)
	others, err := db.OtherConnections(ctx, to)
	if err != nil {
		return out, err
	}
	if others > 0 {
		return out, fmt.Errorf("%d other connections are using this database; stop pwikit before importing", others)
	}

	staging, drop, err := stagingDatabase(ctx, opts)
	if err != nil {
		return out, err
	}
	defer drop()

	report(opts.Report, "unpacking the backup")
	if _, err := Restore(ctx, name, RestoreOptions{DSN: staging, Force: true}); err != nil {
		return out, fmt.Errorf("unpack the backup: %w", err)
	}
	from, err := pgx.Connect(ctx, staging)
	if err != nil {
		return out, err
	}
	defer from.Close(ctx)

	sites := opts.Sites
	if len(sites) == 0 {
		if sites, err = siteSlugs(ctx, from, ""); err != nil {
			return out, err
		}
	}
	if len(sites) != 1 && (opts.Slug != "" || opts.Domain != "" || opts.MediaDomain != "") {
		return out, errors.New("a new slug or domain can only be given when one site is imported")
	}

	for _, slug := range sites {
		report(opts.Report, "importing "+slug)
		done, err := importSite(ctx, from, to, slug, opts)
		if err != nil {
			return out, fmt.Errorf("import %s: %w", slug, err)
		}
		if opts.Storage != nil {
			report(opts.Report, "copying the files of "+slug)
			if done.Files, err = importFiles(ctx, name, opts.Storage, slug, done); err != nil {
				return out, fmt.Errorf("copy the files of %s: %w", slug, err)
			}
		}
		out.Sites = append(out.Sites, done)
	}
	return out, nil
}

func stagingDatabase(ctx context.Context, opts ImportOptions) (string, func(), error) {
	if opts.Staging != "" {
		return opts.Staging, func() {}, nil
	}
	var tag [4]byte
	if _, err := rand.Read(tag[:]); err != nil {
		return "", nil, err
	}
	name := "pwikit_import_" + hex.EncodeToString(tag[:])
	admin, err := pgx.Connect(ctx, opts.DSN)
	if err != nil {
		return "", nil, err
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+db.QuoteName(name)); err != nil {
		admin.Close(ctx)
		return "", nil, fmt.Errorf("make a database to unpack the backup into: %w\n"+
			"  Create an empty database yourself and pass it with -staging", err)
	}
	dsn, err := withDatabase(opts.DSN, name)
	if err != nil {
		admin.Exec(ctx, `DROP DATABASE `+db.QuoteName(name))
		admin.Close(ctx)
		return "", nil, err
	}
	return dsn, func() {
		admin.Exec(context.WithoutCancel(ctx), `DROP DATABASE IF EXISTS `+db.QuoteName(name)+` WITH (FORCE)`)
		admin.Close(context.WithoutCancel(ctx))
	}, nil
}

func withDatabase(dsn, name string) (string, error) {
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		u.Path = "/" + name
		return u.String(), nil
	}
	return dsn + " dbname=" + name, nil
}

type reference struct {
	column, target string
}

// The tables a site's rows live in, with every single-column reference each
// one makes, read from the live schema so a later migration needs no edit here.
func references(ctx context.Context, conn *pgx.Conn) (map[string][]reference, error) {
	rows, err := conn.Query(ctx, `
SELECT c.relname, a.attname, rc.relname
FROM pg_constraint k
JOIN pg_class c ON c.oid = k.conrelid
JOIN pg_class rc ON rc.oid = k.confrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = k.conrelid AND a.attnum = k.conkey[1]
WHERE k.contype = 'f' AND n.nspname = 'public' AND array_length(k.conkey, 1) = 1`)
	if err != nil {
		return nil, fmt.Errorf("read the references: %w", err)
	}
	defer rows.Close()
	out := map[string][]reference{}
	for rows.Next() {
		var table, column, target string
		if err := rows.Scan(&table, &column, &target); err != nil {
			return nil, err
		}
		out[table] = append(out[table], reference{column, target})
	}
	return out, rows.Err()
}

func columns(ctx context.Context, conn *pgx.Conn, table string) ([]string, error) {
	rows, err := conn.Query(ctx, `
SELECT a.attname FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname = $1 AND a.attnum > 0 AND NOT a.attisdropped AND a.attgenerated = ''
ORDER BY a.attnum`, table)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func importSite(ctx context.Context, from, to *pgx.Conn, slug string, opts ImportOptions) (ImportedSite, error) {
	done := ImportedSite{From: slug, Slug: slug}
	if opts.Slug != "" {
		done.Slug = opts.Slug
	}
	tables, err := db.BackupTables(ctx, to)
	if err != nil {
		return done, err
	}
	refs, err := references(ctx, to)
	if err != nil {
		return done, err
	}

	tx, err := to.Begin(ctx)
	if err != nil {
		return done, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	moving := map[string][]string{}
	for _, table := range tables {
		query, err := db.SiteExportQuery(ctx, from, table, true)
		if err != nil {
			return done, err
		}
		if query == "" || db.SiteScopeShared(table) {
			continue
		}
		cols, err := columns(ctx, to, table)
		if err != nil {
			return done, err
		}
		temp := "imp_" + table
		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE `+db.QuoteName(temp)+` ON COMMIT DROP AS SELECT `+
			quoteAll(cols)+` FROM `+db.QuoteName(table)+` WITH NO DATA`); err != nil {
			return done, fmt.Errorf("stage %s: %w", table, err)
		}
		literal, err := quoteLiteral(ctx, from, slug)
		if err != nil {
			return done, err
		}
		var buf bytes.Buffer
		if _, err := from.PgConn().CopyTo(ctx, &buf, `COPY (`+strings.ReplaceAll(query, "$1", literal)+`) TO STDOUT`); err != nil {
			return done, fmt.Errorf("read %s from the backup: %w", table, err)
		}
		tag, err := tx.Conn().PgConn().CopyFrom(ctx, &buf, `COPY `+db.QuoteName(temp)+` FROM STDIN`)
		if err != nil {
			return done, fmt.Errorf("stage %s: %w", table, err)
		}
		done.Rows += tag.RowsAffected()
		moving[table] = cols
	}
	if len(moving) == 0 {
		return done, fmt.Errorf("the backup has no site %q", slug)
	}
	if err := checkSiteFree(ctx, tx, done.Slug, opts); err != nil {
		return done, err
	}

	if err := numberAgain(ctx, tx, moving, &done); err != nil {
		return done, err
	}
	// A page's stored name is unique across the instance and names the folder
	// its files sit in, so one already taken here gets a fresh name.
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE map_media ON COMMIT DROP AS
SELECT media_name AS old, CASE WHEN EXISTS (SELECT 1 FROM web_article a WHERE a.media_name = i.media_name)
THEN gen_random_uuid()::text ELSE media_name END AS new FROM imp_web_article i`); err != nil {
		return done, err
	}
	rows, err := tx.Query(ctx, `SELECT old, new FROM map_media`)
	if err != nil {
		return done, err
	}
	done.media = map[string]string{}
	for rows.Next() {
		var old, renamed string
		if err := rows.Scan(&old, &renamed); err != nil {
			rows.Close()
			return done, err
		}
		done.media[old] = renamed
	}
	rows.Close()
	keys, err := db.LiftForeignKeys(ctx, tx)
	if err != nil {
		return done, err
	}
	names := make([]string, 0, len(moving))
	for table := range moving {
		names = append(names, table)
	}
	sort.Strings(names)
	for _, table := range names {
		if err := insertMoved(ctx, tx, table, moving[table], refs[table], moving, opts); err != nil {
			return done, err
		}
	}
	if err := db.RestoreForeignKeys(ctx, tx, keys); err != nil {
		return done, err
	}
	return done, tx.Commit(ctx)
}

func quoteAll(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = db.QuoteName(c)
	}
	return strings.Join(quoted, ", ")
}

func quoteLiteral(ctx context.Context, conn *pgx.Conn, value string) (string, error) {
	var quoted string
	err := conn.QueryRow(ctx, `SELECT quote_literal($1::text)`, value).Scan(&quoted)
	return quoted, err
}

func checkSiteFree(ctx context.Context, tx pgx.Tx, slug string, opts ImportOptions) error {
	var taken string
	err := tx.QueryRow(ctx, `
SELECT CASE WHEN s.slug = $1 THEN 'slug ' || s.slug ELSE 'domain ' || s.domain END
FROM web_site s, imp_web_site i
WHERE s.slug = $1 OR s.domain = COALESCE(NULLIF($2, ''), i.domain)
   OR s.media_domain = COALESCE(NULLIF($3, ''), i.media_domain)
LIMIT 1`, slug, opts.Domain, opts.MediaDomain).Scan(&taken)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("a site here already has that %s; give the imported one another with -slug, -domain or -media-domain", taken)
}

// New ids come from this database's own counters, and the old ones are kept
// beside them so every reference can be rewritten through the map.
func numberAgain(ctx context.Context, tx pgx.Tx, moving map[string][]string, done *ImportedSite) error {
	for table, cols := range moving {
		if !contains(cols, "id") {
			continue
		}
		mapName := db.QuoteName("map_" + table)
		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE `+mapName+` (old bigint PRIMARY KEY, new bigint NOT NULL, kept boolean NOT NULL DEFAULT false) ON COMMIT DROP`); err != nil {
			return err
		}
		if table == "web_user" {
			if err := matchAccounts(ctx, tx, done); err != nil {
				return err
			}
		}
		var sequence *string
		if err := tx.QueryRow(ctx, `SELECT pg_get_serial_sequence($1, 'id')`, "public."+table).Scan(&sequence); err != nil {
			return err
		}
		next := `(SELECT COALESCE(MAX(id), 0) FROM ` + db.QuoteName(table) + `) + row_number() OVER (ORDER BY i.id)`
		if sequence != nil {
			next = `nextval(` + quoteText(*sequence) + `)`
		}
		if _, err := tx.Exec(ctx, `INSERT INTO `+mapName+` (old, new)
SELECT i.id, `+next+` FROM `+db.QuoteName("imp_"+table)+` i
WHERE NOT EXISTS (SELECT 1 FROM `+mapName+` m WHERE m.old = i.id)
ORDER BY i.id`); err != nil {
			return fmt.Errorf("number %s again: %w", table, err)
		}
	}
	return nil
}

func quoteText(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// Only a Wikidot account or an email address proves two accounts are one
// person. A name alone could belong to two different people.
func matchAccounts(ctx context.Context, tx pgx.Tx, done *ImportedSite) error {
	tag, err := tx.Exec(ctx, `
INSERT INTO map_web_user (old, new, kept)
SELECT DISTINCT ON (i.id) i.id, u.id, true
FROM imp_web_user i
JOIN web_user u ON (i.wikidot_username IS NOT NULL AND i.wikidot_username <> '' AND lower(u.wikidot_username) = lower(i.wikidot_username))
   OR (i.email <> '' AND lower(u.email) = lower(i.email))
ORDER BY i.id, u.id`)
	if err != nil {
		return fmt.Errorf("match the accounts: %w", err)
	}
	done.SameAccounts = int(tag.RowsAffected())
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM imp_web_user`).Scan(&total); err != nil {
		return err
	}
	done.NewAccounts = total - done.SameAccounts
	return nil
}

func insertMoved(ctx context.Context, tx pgx.Tx, table string, cols []string, refs []reference, moving map[string][]string, opts ImportOptions) error {
	exprs := make([]string, len(cols))
	targets := map[string]string{}
	for _, r := range refs {
		if _, moved := moving[r.target]; moved && contains(moving[r.target], "id") {
			targets[r.column] = r.target
		}
	}
	for i, c := range cols {
		q := "i." + db.QuoteName(c)
		switch {
		case c == "id" && contains(cols, "id"):
			exprs[i] = `(SELECT new FROM ` + db.QuoteName("map_"+table) + ` WHERE old = i.id)`
		case targets[c] != "":
			exprs[i] = `(SELECT new FROM ` + db.QuoteName("map_"+targets[c]) + ` WHERE old = ` + q + `)`
		default:
			exprs[i] = q
		}
	}
	where := ""
	switch table {
	case "web_user":
		where = ` WHERE NOT EXISTS (SELECT 1 FROM map_web_user m WHERE m.old = i.id AND m.kept)`
		for i, c := range cols {
			if c == "username" {
				// A name can be held here by someone else, so the newcomer gets
				// the name with its new number after it.
				exprs[i] = `CASE WHEN EXISTS (SELECT 1 FROM web_user u WHERE lower(u.username) = lower(i.username))
THEN i.username || '-' || (SELECT new FROM map_web_user WHERE old = i.id) ELSE i.username END`
			}
		}
	case "dynamic_preferences_users_userpreferencemodel":
		where = ` WHERE NOT EXISTS (SELECT 1 FROM map_web_user m WHERE m.old = i.instance_id AND m.kept)`
	case "web_article":
		for i, c := range cols {
			if c == "media_name" {
				exprs[i] = `(SELECT new FROM map_media WHERE old = i.media_name)`
			}
		}
	case "web_site":
		for i, c := range cols {
			switch {
			case c == "slug" && opts.Slug != "":
				exprs[i] = quoteText(opts.Slug)
			case c == "domain" && opts.Domain != "":
				exprs[i] = quoteText(opts.Domain)
			case c == "media_domain" && opts.MediaDomain != "":
				exprs[i] = quoteText(opts.MediaDomain)
			}
		}
	}
	var identity bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname = $1 AND a.attidentity = 'a')`, table).Scan(&identity); err != nil {
		return err
	}
	override := ""
	if identity {
		override = " OVERRIDING SYSTEM VALUE"
	}
	_, err := tx.Exec(ctx, `INSERT INTO `+db.QuoteName(table)+` (`+quoteAll(cols)+`)`+override+` SELECT `+
		strings.Join(exprs, ", ")+` FROM `+db.QuoteName("imp_"+table)+` i`+where)
	if err != nil {
		return fmt.Errorf("write %s: %w", table, err)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Only the files the imported rows point at come along. The stored names are
// random, so they keep their keys; only a theme moves under the new slug.
func importFiles(ctx context.Context, archive string, set *storage.Set, oldSlug string, done ImportedSite) (int, error) {
	newSlug := done.Slug
	folders := map[string]string{}
	for old, renamed := range done.media {
		folders[media.QuoteName(old)] = media.QuoteName(renamed)
	}

	file, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	zip, err := gzip.NewReader(file)
	if err != nil {
		return 0, err
	}
	defer zip.Close()
	site, shared := set.For(newSlug).Store, set.Shared.Store
	put := 0
	tr := tar.NewReader(zip)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			return put, nil
		}
		if err != nil {
			return put, err
		}
		key, ok := strings.CutPrefix(head.Name, filesDir+"/")
		if !ok || storage.CheckKey(key) != nil {
			continue
		}
		var store storage.Store
		parts := strings.SplitN(key, "/", 3)
		switch {
		case len(parts) == 3 && parts[0] == "media" && folders[parts[1]] != "":
			key, store = storage.Join("media", folders[parts[1]], parts[2]), site
		case strings.HasPrefix(key, "theme/"+oldSlug+"/"):
			key, store = "theme/"+newSlug+"/"+strings.TrimPrefix(key, "theme/"+oldSlug+"/"), site
		case storage.IsShared(key) && !storage.Exists(ctx, shared, key):
			store = shared
		default:
			continue
		}
		if err := store.Put(ctx, key, tr, head.Size); err != nil {
			return put, fmt.Errorf("put %s: %w", key, err)
		}
		put++
	}
}
