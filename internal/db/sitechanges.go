package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SiteChangeFilter struct {
	Hidden []string

	Types []string

	Category    string
	HasCategory bool

	HasUser    bool
	UserIDs    []int64
	WithSystem bool
}

type SiteChange struct {
	RevNumber int
	Type      string
	Meta      []byte
	Comment   string
	CreatedAt time.Time
	UserID    *int64

	ArticleTitle    string
	ArticleCategory string
	ArticleName     string
}

// A revert carries the types it undid in meta rather than in the type column,
// so asking for one type has to reach both places.
func (f SiteChangeFilter) build(b *listBuilder) string {
	if len(f.Hidden) > 0 {
		b.where = append(b.where, "NOT (a.category = ANY("+b.arg(f.Hidden)+"))")
	}
	if len(f.Types) > 0 {
		ors := []string{"e.type = ANY(" + b.arg(f.Types) + ")"}
		for _, t := range f.Types {
			ors = append(ors, "(e.meta -> 'subtypes') @> "+b.arg(strconv.Quote(t))+"::jsonb")
		}
		b.where = append(b.where, "("+strings.Join(ors, " OR ")+")")
	}
	if f.HasCategory {
		b.where = append(b.where, "a.category = "+b.arg(f.Category))
	}
	if f.HasUser {
		or := "e.user_id = ANY(" + b.arg(f.UserIDs) + ")"
		if f.WithSystem {
			or += " OR e.user_id IS NULL"
		}
		b.where = append(b.where, "("+or+")")
	}

	sql := "FROM web_articlelogentry e\nJOIN web_article a ON a.id = e.article_id"
	if len(b.where) > 0 {
		sql += "\nWHERE " + strings.Join(b.where, "\n  AND ")
	}
	return sql
}

func (f SiteChangeFilter) selectSQL(b *listBuilder, offset, limit int) string {
	sql := `SELECT e.rev_number, e.type, e.meta, e.comment, e.created_at, e.user_id,
       a.title, a.category, a.name
` + f.build(b) + "\nORDER BY e.created_at DESC" +
		"\nLIMIT " + b.arg(limit) + "\nOFFSET " + b.arg(offset)
	return sql
}

// Exposed so the schema-drift test can send a built statement to Postgres the
// way it sends the ones written out by hand.
func (f SiteChangeFilter) SelectSQL(offset, limit int) (string, []any) {
	b := &listBuilder{}
	return f.selectSQL(b, offset, limit), b.args
}

func (d *DB) SiteChanges(ctx context.Context, f SiteChangeFilter, offset, limit int) ([]SiteChange, error) {
	b := &listBuilder{}
	sql := f.selectSQL(b, offset, limit)

	rows, err := d.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("query site changes: %w", err)
	}
	defer rows.Close()

	var out []SiteChange
	for rows.Next() {
		var c SiteChange
		if err := rows.Scan(&c.RevNumber, &c.Type, &c.Meta, &c.Comment, &c.CreatedAt,
			&c.UserID, &c.ArticleTitle, &c.ArticleCategory, &c.ArticleName); err != nil {
			return nil, fmt.Errorf("scan site change: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read site changes: %w", err)
	}
	return out, nil
}

func (d *DB) SiteChangeCount(ctx context.Context, f SiteChangeFilter) (int, error) {
	b := &listBuilder{}
	sql := "SELECT COUNT(*)\n" + f.build(b)

	var n int
	if err := d.pool.QueryRow(ctx, sql, b.args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count site changes: %w", err)
	}
	return n, nil
}

var qArticleCategories = register("ArticleCategories", `
SELECT DISTINCT category
FROM web_article
WHERE NOT (category = ANY($1))`)

func (d *DB) ArticleCategories(ctx context.Context, hidden []string) ([]string, error) {
	if hidden == nil {
		hidden = []string{}
	}
	rows, err := d.pool.Query(ctx, qArticleCategories, hidden)
	if err != nil {
		return nil, fmt.Errorf("query article categories: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan article category: %w", err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read article categories: %w", err)
	}
	return out, nil
}

var qUserIDsByName = register("UserIDsByName", `
SELECT id
FROM web_user
WHERE username = $1 OR wikidot_username = $1`)

// Django uppercases both sides rather than leaning on citext, and a wiki that
// renamed the column to text would silently start matching case-sensitively.
var qUserIDsByNamePart = register("UserIDsByNamePart", `
SELECT id
FROM web_user
WHERE UPPER(username::text) LIKE UPPER($1) OR UPPER(wikidot_username::text) LIKE UPPER($1)`)

func (d *DB) UserIDsByName(ctx context.Context, name string, partial bool) ([]int64, error) {
	sql, arg := qUserIDsByName, name
	if partial {
		sql, arg = qUserIDsByNamePart, likeContains(name)
	}
	rows, err := d.pool.Query(ctx, sql, arg)
	if err != nil {
		return nil, fmt.Errorf("query users named %q: %w", name, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user named %q: %w", name, err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read users named %q: %w", name, err)
	}
	return out, nil
}

var qUsersByIDs = register("UsersByIDs", `
SELECT `+userColumns+`
FROM web_user
WHERE id = ANY($1)`)

func (d *DB) UsersByIDs(ctx context.Context, ids []int64) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := d.pool.Query(ctx, qUsersByIDs, ids)
	if err != nil {
		return nil, fmt.Errorf("query users by id: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		dest, finish := userDest(&u)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan user by id: %w", err)
		}
		finish()
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read users by id: %w", err)
	}
	return out, nil
}

func likeContains(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}
