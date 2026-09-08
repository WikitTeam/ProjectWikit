package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LogWikidot marks a revision an import carried over whose kind this engine has
// no word for, such as a rename Wikidot recorded without keeping the source.
const LogWikidot = "wikidot"

type ImportUser struct {
	WikidotID   int64
	Username    string
	DisplayName string
}

var (
	qUsersByWikidotID = register("UsersByWikidotID", `
SELECT wikidot_user_id, id FROM web_user WHERE wikidot_user_id = ANY($1)`)

	qUsersByWikidotName = register("UsersByWikidotName", `
SELECT wikidot_username, id FROM web_user WHERE wikidot_username = ANY($1)`)

	qAdoptWikidotID = register("AdoptWikidotID", `
UPDATE web_user SET wikidot_user_id = $2 WHERE id = $1 AND wikidot_user_id IS NULL`)

	qInsertWikidotUser = register("InsertWikidotUser", `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, wikidot_username, wikidot_user_id, display_name, type, bio,
	is_forum_active, is_active, can_send_direct_messages, pending_email, previous_email)
VALUES ('!', false, '', '', '', $4, $5, $1, $2, $3, 'wikidot', '', true, false, true, '', '')
RETURNING id`)
)

// EnsureWikidotUsers answers the local id of every account the archive names,
// creating the ones this database has never seen. Nothing already here is
// changed, so importing a second archive that overlaps adds rows and edits none.
func (d *DB) EnsureWikidotUsers(ctx context.Context, users []ImportUser, at time.Time) (map[int64]int64, error) {
	out := make(map[int64]int64, len(users))
	if len(users) == 0 {
		return out, nil
	}

	ids := make([]int64, 0, len(users))
	names := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.WikidotID)
		if u.Username != "" {
			names = append(names, u.Username)
		}
	}

	rows, err := d.pool.Query(ctx, qUsersByWikidotID, ids)
	if err != nil {
		return nil, fmt.Errorf("look up imported users by id: %w", err)
	}
	for rows.Next() {
		var wikidotID, local int64
		if err := rows.Scan(&wikidotID, &local); err != nil {
			rows.Close()
			return nil, err
		}
		out[wikidotID] = local
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// An account the previous importer created carries the name but not the
	// number, so the number is written onto it rather than a second row.
	byName := map[string]int64{}
	if len(names) > 0 {
		rows, err := d.pool.Query(ctx, qUsersByWikidotName, names)
		if err != nil {
			return nil, fmt.Errorf("look up imported users by name: %w", err)
		}
		for rows.Next() {
			var name string
			var local int64
			if err := rows.Scan(&name, &local); err != nil {
				rows.Close()
				return nil, err
			}
			byName[lowered(name)] = local
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	for _, u := range users {
		if _, ok := out[u.WikidotID]; ok {
			continue
		}
		if local, ok := byName[lowered(u.Username)]; ok {
			if _, err := d.pool.Exec(ctx, qAdoptWikidotID, local, u.WikidotID); err != nil {
				return nil, fmt.Errorf("adopt wikidot id %d: %w", u.WikidotID, err)
			}
			out[u.WikidotID] = local
			continue
		}
		name := u.Username
		if name == "" {
			name = fmt.Sprintf("deleted-%d", u.WikidotID)
		}
		local, err := d.insertWikidotUser(ctx, u.WikidotID, name, u.DisplayName, at)
		if err != nil {
			return nil, err
		}
		out[u.WikidotID] = local
		byName[lowered(name)] = local
	}
	return out, nil
}

func (d *DB) insertWikidotUser(ctx context.Context, wikidotID int64, name, display string, at time.Time) (int64, error) {
	local, err := mediaName()
	if err != nil {
		return 0, err
	}
	var id int64
	err = d.pool.QueryRow(ctx, qInsertWikidotUser, name, wikidotID, display, at, local).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create the imported account %q: %w", name, err)
	}
	return id, nil
}

type ImportRevision struct {
	Number  int
	UserID  *int64
	Comment string
	At      time.Time

	// Source is nil for a revision Wikidot recorded without keeping the text,
	// which is most of them once a page has been renamed or retagged.
	Source *string
	IsNew  bool
}

type ImportVote struct {
	UserID int64
	Rate   float64
}

type ImportArticle struct {
	Category  string
	Name      string
	Title     string
	Locked    bool
	CreatedAt time.Time
	UpdatedAt time.Time
	AuthorID  *int64

	Revisions []ImportRevision
	Votes     []ImportVote
	TagIDs    []int64
}

var (
	qImportArticle = register("ImportArticle", `
INSERT INTO web_article (site_id, category, name, title, locked, created_at, updated_at, media_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id`)

	qImportLogEntry = register("ImportLogEntry", `
INSERT INTO web_articlelogentry (article_id, user_id, type, meta, comment, created_at, rev_number)
VALUES ($1, $2, $3, $4, $5, $6, $7)`)

	qImportVote = register("ImportVote", `
INSERT INTO web_vote (article_id, user_id, rate, date)
VALUES ($1, $2, $3, NULL)
ON CONFLICT DO NOTHING`)

	qImportArticleTag = register("ImportArticleTag", `
INSERT INTO web_article_tags (article_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`)
)

// ImportArticle writes one page and its whole history as the archive recorded
// it. It goes around the ordinary save path on purpose, since that one renumbers
// revisions, rebuilds links and tells subscribers about every one of them.
func (d *DB) ImportArticle(ctx context.Context, siteID int64, a ImportArticle) (int64, error) {
	media, err := mediaName()
	if err != nil {
		return 0, err
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin importing %q: %w", a.Name, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	var id int64
	err = tx.QueryRow(ctx, qImportArticle, siteID, a.Category, a.Name, a.Title, a.Locked,
		a.CreatedAt, a.UpdatedAt, media).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("import article %q: %w", a.Name, err)
	}
	if a.AuthorID != nil {
		if _, err := tx.Exec(ctx, qInsertArticleAuthor, id, *a.AuthorID); err != nil {
			return 0, fmt.Errorf("credit the author of %q: %w", a.Name, err)
		}
	}

	for _, rev := range a.Revisions {
		kind := LogWikidot
		meta := map[string]any{}
		if rev.Source != nil {
			var versionID int64
			err := tx.QueryRow(ctx, qInsertArticleVersion, id, *rev.Source, rev.At).Scan(&versionID)
			if err != nil {
				return 0, fmt.Errorf("import a version of %q: %w", a.Name, err)
			}
			kind = LogSource
			meta["version_id"] = versionID
			if rev.IsNew {
				kind = LogNew
				meta["title"] = a.Title
			}
		}
		encoded, err := json.Marshal(meta)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(ctx, qImportLogEntry, id, rev.UserID, kind, encoded, rev.Comment, rev.At, rev.Number)
		if err != nil {
			return 0, fmt.Errorf("import a revision of %q: %w", a.Name, err)
		}
	}

	for _, vote := range a.Votes {
		if _, err := tx.Exec(ctx, qImportVote, id, vote.UserID, vote.Rate); err != nil {
			return 0, fmt.Errorf("import a vote on %q: %w", a.Name, err)
		}
	}
	for _, tagID := range a.TagIDs {
		if _, err := tx.Exec(ctx, qImportArticleTag, id, tagID); err != nil {
			return 0, fmt.Errorf("tag %q: %w", a.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit %q: %w", a.Name, err)
	}
	return id, nil
}

var qImportParent = register("ImportParent", `
UPDATE web_article SET parent_id = $2 WHERE id = $1 AND site_id = $3`)

func (d *DB) SetImportedParent(ctx context.Context, siteID, articleID, parentID int64) error {
	if _, err := d.pool.Exec(ctx, qImportParent, articleID, parentID, siteID); err != nil {
		return fmt.Errorf("set the parent of %d: %w", articleID, err)
	}
	return nil
}

func lowered(s string) string { return strings.ToLower(s) }

// EnsureTags resolves the tag names an archive carries, creating the ones this
// site has never had. It is the import's way in to the same resolution the
// editor uses, so a tag written here is the tag the editor would have made.
func (d *DB) EnsureTags(ctx context.Context, siteID int64, names []string, allowCreate bool) ([]int64, error) {
	if len(names) == 0 {
		return nil, nil
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin resolving tags: %w", err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	tags, err := resolveTags(ctx, tx, siteID, names, allowCreate)
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(tags))
	for _, tag := range tags {
		out = append(out, tag.ID)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tags: %w", err)
	}
	return out, nil
}
