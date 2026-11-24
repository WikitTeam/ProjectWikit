package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

type fileSet map[string]storage.Store

var qAttachments = `
SELECT s.slug, a.media_name, f.media_name
FROM web_file f
JOIN web_article a ON a.id = f.article_id
JOIN web_site s ON s.id = a.site_id
WHERE $1 = '' OR s.slug = $1`

// Attachments are listed from the table, so nothing a failed upload left behind
// comes along. Resized copies are left out since they are made on demand.
func collectFiles(ctx context.Context, conn *pgx.Conn, set *storage.Set, site string, opts CreateOptions) (fileSet, error) {
	files := fileSet{}
	rows, err := conn.Query(ctx, qAttachments, site)
	if err != nil {
		return nil, fmt.Errorf("list the attachments: %w", err)
	}
	for rows.Next() {
		var slug, article, name string
		if err := rows.Scan(&slug, &article, &name); err != nil {
			rows.Close()
			return nil, err
		}
		files[storage.Join("media", media.QuoteName(article), media.QuoteName(name))] = set.For(slug).Store
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	slugs, err := siteSlugs(ctx, conn, site)
	if err != nil {
		return nil, err
	}
	for _, slug := range slugs {
		store := set.For(slug).Store
		if err := store.Walk(ctx, "theme/"+slug, func(key string, _ storage.Info) error {
			files[key] = store
			return nil
		}); err != nil {
			return nil, fmt.Errorf("list the themes of %s: %w", slug, err)
		}
	}

	shared := set.Shared.Store
	if site == "" {
		err := shared.Walk(ctx, "-", func(key string, _ storage.Info) error {
			files[key] = shared
			return nil
		})
		return files, err
	}
	refs, err := sharedRefs(ctx, conn, site, opts.KeepPasswords)
	if err != nil {
		return nil, err
	}
	for _, key := range refs {
		if storage.CheckKey(key) == nil {
			files[key] = shared
		}
	}
	return files, nil
}

func siteSlugs(ctx context.Context, conn *pgx.Conn, site string) ([]string, error) {
	rows, err := conn.Query(ctx, `SELECT slug FROM web_site WHERE $1 = '' OR slug = $1 ORDER BY slug`, site)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// Avatars and icons live outside any one site, so a site's export takes only
// the ones its own rows name.
func sharedRefs(ctx context.Context, conn *pgx.Conn, site string, keepPasswords bool) ([]string, error) {
	people, err := db.SiteExportQuery(ctx, conn, "web_user", keepPasswords)
	if err != nil {
		return nil, err
	}
	people = strings.ReplaceAll(people, "$1", "$2")
	query := `
SELECT avatar FROM (` + people + `) u WHERE avatar LIKE '-/%'
UNION SELECT auth_icon FROM web_site WHERE slug = $1 AND auth_icon LIKE '-/%'
UNION SELECT r.icon FROM web_role r JOIN web_site s ON s.id = r.site_id WHERE s.slug = $1 AND r.icon LIKE '-/%'`
	rows, err := conn.Query(ctx, query, site, site)
	if err != nil {
		return nil, fmt.Errorf("list the avatars and icons of %s: %w", site, err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// A row whose file has gone missing is reported and left out rather than
// failing a backup that is otherwise whole.
func hashFiles(ctx context.Context, files fileSet, m *Manifest, say func(string)) error {
	m.Files.Included = true
	for key, store := range files {
		f, err := store.Open(ctx, key)
		if errors.Is(err, storage.ErrNotExist) {
			report(say, "missing file "+key)
			delete(files, key)
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", key, err)
		}
		sum := sha256.New()
		n, err := io.Copy(sum, f)
		f.Close()
		if err != nil {
			return fmt.Errorf("read %s: %w", key, err)
		}
		m.Files.Entries[key] = hex.EncodeToString(sum.Sum(nil))
		m.Files.Count++
		m.Files.Bytes += n
	}
	return nil
}

// Each key goes back to the store that holds that kind of file for that site,
// which the restored tables name.
type Router struct {
	set      *storage.Set
	articles map[string]string
}

func NewRouter(ctx context.Context, conn *pgx.Conn, set *storage.Set) (*Router, error) {
	rows, err := conn.Query(ctx, `SELECT a.media_name, s.slug FROM web_article a JOIN web_site s ON s.id = a.site_id`)
	if err != nil {
		return nil, fmt.Errorf("read where the pages live: %w", err)
	}
	defer rows.Close()
	r := &Router{set: set, articles: map[string]string{}}
	for rows.Next() {
		var article, slug string
		if err := rows.Scan(&article, &slug); err != nil {
			return nil, err
		}
		r.articles[media.QuoteName(article)] = slug
	}
	return r, rows.Err()
}

// A file shared by the instance belongs to no site and answers "".
func (r *Router) Site(key string) string {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 2 {
		return ""
	}
	switch parts[0] {
	case "media", "resized":
		return r.articles[parts[1]]
	case "theme":
		return parts[1]
	}
	return ""
}

func (r *Router) Backend(key string) *storage.Backend {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 2 {
		return r.set.Shared
	}
	switch parts[0] {
	case "media", "resized":
		return r.set.For(r.articles[parts[1]])
	case "theme":
		return r.set.For(parts[1])
	}
	return r.set.Shared
}
