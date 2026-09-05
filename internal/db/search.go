package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type SearchFilter struct {
	Words    []string
	Category string
	AuthorID *int64
	Include  [][]int64
	Exclude  []int64
	From     *time.Time
	To       *time.Time
	Hidden   []string
}

func (f SearchFilter) where(b *listBuilder) string {
	parts := []string{"si.article_id IS NOT NULL"}
	if len(f.Hidden) > 0 {
		parts = append(parts, "NOT (a.category = ANY("+b.arg(f.Hidden)+"))")
	}
	if f.Category != "" {
		parts = append(parts, "a.category = "+b.arg(strings.ToLower(f.Category)))
	}
	if f.AuthorID != nil {
		parts = append(parts, "EXISTS (SELECT 1 FROM web_article_authors aa"+
			" WHERE aa.article_id = a.id AND aa.user_id = "+b.arg(*f.AuthorID)+")")
	}
	// One name can name several tags and a page needs only one of them. Naming
	// two tags asks for a page carrying both.
	for _, group := range f.Include {
		parts = append(parts, "EXISTS (SELECT 1 FROM web_article_tags at"+
			" WHERE at.article_id = a.id AND at.tag_id = ANY("+b.arg(group)+"))")
	}
	if len(f.Exclude) > 0 {
		parts = append(parts, "NOT EXISTS (SELECT 1 FROM web_article_tags at"+
			" WHERE at.article_id = a.id AND at.tag_id = ANY("+b.arg(f.Exclude)+"))")
	}
	if f.From != nil {
		parts = append(parts, "a.created_at >= "+b.arg(*f.From))
	}
	if f.To != nil {
		parts = append(parts, "a.created_at <= "+b.arg(*f.To))
	}
	for _, word := range f.Words {
		parts = append(parts, "si.content_plaintext ILIKE "+b.arg(likeContains(word)))
	}
	return "FROM web_articlesearchindex si\nJOIN web_article a ON a.id = si.article_id\nWHERE " +
		strings.Join(parts, "\n  AND ")
}

type SearchHit struct {
	Article   Article
	Plaintext string
}

func (f SearchFilter) SelectSQL(offset, limit int) (string, []any) {
	b := &listBuilder{}
	return f.selectSQL(b, offset, limit), b.args
}

func (f SearchFilter) selectSQL(b *listBuilder, offset, limit int) string {
	return "SELECT " + prefixedArticleColumns + ", si.content_plaintext\n" +
		f.where(b) + "\nORDER BY a.created_at DESC, a.id DESC" +
		"\nLIMIT " + b.arg(limit) + "\nOFFSET " + b.arg(offset)
}

func (d *DB) SearchArticles(ctx context.Context, f SearchFilter, offset, limit int) ([]SearchHit, error) {
	b := &listBuilder{}
	sql := f.selectSQL(b, offset, limit)

	rows, err := d.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("query search: %w", err)
	}
	defer rows.Close()

	var out []SearchHit
	for rows.Next() {
		var hit SearchHit
		a := &hit.Article
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID, &a.Locked,
			&a.CreatedAt, &a.UpdatedAt, &a.MediaName, &hit.Plaintext); err != nil {
			return nil, fmt.Errorf("scan search hit: %w", err)
		}
		out = append(out, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search: %w", err)
	}
	return out, nil
}

func (f SearchFilter) CountSQL() (string, []any) {
	b := &listBuilder{}
	return "SELECT COUNT(*)\n" + f.where(b), b.args
}

func (d *DB) SearchCount(ctx context.Context, f SearchFilter) (int, error) {
	sql, args := f.CountSQL()

	var n int
	if err := d.pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count search: %w", err)
	}
	return n, nil
}

var qTagIDsByFullName = register("TagIDsByFullName", `
SELECT t.id
FROM web_tag t
JOIN web_tagscategory c ON c.id = t.category_id
WHERE t.name = $1 AND ($2 = '' OR c.slug = $2)`)

// A name without a category matches that tag in every category, which is what
// makes one name able to name several tags.
func (d *DB) TagIDsByFullName(ctx context.Context, category, name string) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qTagIDsByFullName, name, category)
	if err != nil {
		return nil, fmt.Errorf("query tag %q: %w", name, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan tag id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

var qAuthorsOfArticles = register("AuthorsOfArticles", `
SELECT link.article_id, `+prefixed("u", userColumns)+`
FROM web_article_authors link
JOIN web_user u ON u.id = link.user_id
WHERE link.article_id = ANY($1)
ORDER BY link.article_id, link.id`)

func (d *DB) AuthorsOfArticles(ctx context.Context, ids []int64) (map[int64][]User, error) {
	out := make(map[int64][]User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qAuthorsOfArticles, ids)
	if err != nil {
		return nil, fmt.Errorf("query authors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var u User
		dest, finish := userDest(&u)
		if err := rows.Scan(append([]any{&id}, dest...)...); err != nil {
			return nil, fmt.Errorf("scan author: %w", err)
		}
		finish()
		out[id] = append(out[id], u)
	}
	return out, rows.Err()
}

var qVoteStatsOfArticles = register("VoteStatsOfArticles", `
SELECT article_id,
       COALESCE(SUM(rate), 0),
       COUNT(rate),
       COUNT(rate) FILTER (WHERE rate = 1),
       COALESCE(AVG(rate), 0),
       COUNT(rate) FILTER (WHERE rate >= 3)
FROM web_vote
WHERE article_id = ANY($1)
GROUP BY article_id`)

func (d *DB) VoteStatsOfArticles(ctx context.Context, ids []int64) (map[int64]VoteStats, error) {
	out := make(map[int64]VoteStats, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qVoteStatsOfArticles, ids)
	if err != nil {
		return nil, fmt.Errorf("query votes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var s VoteStats
		if err := rows.Scan(&id, &s.Sum, &s.Count, &s.GoodUpDown, &s.Average, &s.GoodStars); err != nil {
			return nil, fmt.Errorf("scan votes: %w", err)
		}
		out[id] = s
	}
	return out, rows.Err()
}

var qCommentCountsOfArticles = register("CommentCountsOfArticles", `
SELECT t.article_id, COUNT(p.id)
FROM web_forumthread t
JOIN web_forumpost p ON p.thread_id = t.id
WHERE t.article_id = ANY($1)
GROUP BY t.article_id`)

func (d *DB) CommentCountsOfArticles(ctx context.Context, ids []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qCommentCountsOfArticles, ids)
	if err != nil {
		return nil, fmt.Errorf("query comment counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scan comment count: %w", err)
		}
		out[id] = n
	}
	return out, rows.Err()
}

var qTagsOfArticles = register("TagsOfArticles", `
SELECT link.article_id, t.id, c.slug, t.name
FROM web_article_tags link
JOIN web_tag t ON t.id = link.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE link.article_id = ANY($1)
ORDER BY link.article_id, c.slug, t.name`)

type ArticleTag struct {
	ID       int64
	Category string
	Name     string
}

func (t ArticleTag) FullName() string {
	if t.Category == DefaultCategory || t.Category == "" {
		return t.Name
	}
	return t.Category + ":" + t.Name
}

func (d *DB) TagsOfArticles(ctx context.Context, ids []int64) (map[int64][]ArticleTag, error) {
	out := make(map[int64][]ArticleTag, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qTagsOfArticles, ids)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var t ArticleTag
		if err := rows.Scan(&id, &t.ID, &t.Category, &t.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out[id] = append(out[id], t)
	}
	return out, rows.Err()
}

var qLatestEditorsOfArticles = register("LatestEditorsOfArticles", `
SELECT DISTINCT ON (e.article_id) e.article_id, `+prefixed("u", userColumns)+`
FROM web_articlelogentry e
JOIN web_user u ON u.id = e.user_id
WHERE e.article_id = ANY($1)
ORDER BY e.article_id, e.rev_number DESC`)

func (d *DB) LatestEditorsOfArticles(ctx context.Context, ids []int64) (map[int64]User, error) {
	out := make(map[int64]User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qLatestEditorsOfArticles, ids)
	if err != nil {
		return nil, fmt.Errorf("query latest editors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var u User
		dest, finish := userDest(&u)
		if err := rows.Scan(append([]any{&id}, dest...)...); err != nil {
			return nil, fmt.Errorf("scan latest editor: %w", err)
		}
		finish()
		out[id] = u
	}
	return out, rows.Err()
}

var qCategoryRatingModes = register("CategoryRatingModes", `
SELECT c.name, s.rating_mode
FROM web_settings s
JOIN web_category c ON c.id = s.category_id`)

func (d *DB) CategoryRatingModes(ctx context.Context) (map[string]string, error) {
	rows, err := d.pool.Query(ctx, qCategoryRatingModes)
	if err != nil {
		return nil, fmt.Errorf("query category rating modes: %w", err)
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var name, mode string
		if err := rows.Scan(&name, &mode); err != nil {
			return nil, fmt.Errorf("scan category rating mode: %w", err)
		}
		out[name] = mode
	}
	return out, rows.Err()
}
