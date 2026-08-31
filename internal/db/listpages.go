package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Range and ExcludeRange read both ends, the rest read only the end the
// operator names.
type TimeFilter struct {
	Op    string
	Start time.Time
	End   time.Time
}

const (
	TimeRange        = "range"
	TimeExcludeRange = "exclude_range"
	TimeLT           = "lt"
	TimeLTE          = "lte"
	TimeGT           = "gt"
	TimeGTE          = "gte"
)

type NumFilter struct {
	Op    string
	Value float64
}

const (
	NumEQ  = "eq"
	NumNE  = "ne"
	NumLT  = "lt"
	NumLTE = "lte"
	NumGT  = "gt"
	NumGTE = "gte"
)

const (
	SortCreatedAt  = "created_at"
	SortCreatedBy  = "created_by"
	SortName       = "name"
	SortTitle      = "title"
	SortUpdatedAt  = "updated_at"
	SortFullName   = "fullname"
	SortRating     = "rating"
	SortVotes      = "votes"
	SortPopularity = "popularity"
	SortRandom     = "random"
)

type Sort struct {
	Column    string
	Ascending bool
}

type ListFilter struct {
	Hidden []string

	PageType      string
	Name          string
	HasName       bool
	NamePrefix    string
	HasNamePrefix bool

	NoTags       bool
	RequiredTags []int64
	PresentTags  []int64
	AbsentTags   []int64
	ExactTags    []int64

	Categories    []string
	NotCategories []string

	HasParent    bool
	ParentID     *int64
	HasNotParent bool
	NotParentID  *int64

	AuthorID *int64

	CreatedAt *TimeFilter

	Rating     *NumFilter
	Votes      *NumFilter
	Popularity *NumFilter

	RatingMode string

	Sort Sort
}

const (
	PageTypeNormal = "normal"
	PageTypeHidden = "hidden"
)

type listBuilder struct {
	args  []any
	where []string
}

func (b *listBuilder) arg(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

// A site that rates nothing still has to sort by something, so the id stands
// in.
func ratingExpr(mode string) string {
	switch mode {
	case "updown":
		return "COALESCE(v.sum_rate, 0.0)"
	case "stars":
		return "COALESCE(v.avg_rate, 0.0)"
	}
	return "a.id"
}

// The rounding is Postgres' own, which breaks ties away from zero and so does
// not agree with the popularity the page variables compute.
func popularityExpr(mode string) string {
	good := "COALESCE(v.good_updown, 0)"
	if mode == "stars" {
		good = "COALESCE(v.good_stars, 0)"
	}
	return "CASE WHEN COALESCE(v.num_votes, 0) > 0 THEN ROUND(" +
		good + "::float / COALESCE(v.num_votes, 0)::float * 100) ELSE 0 END"
}

const votesExpr = "COALESCE(v.num_votes, 0)"

const voteJoin = `
LEFT JOIN (
	SELECT article_id,
	       COUNT(*) AS num_votes,
	       COALESCE(SUM(rate), 0.0) AS sum_rate,
	       COALESCE(AVG(rate), 0.0) AS avg_rate,
	       COUNT(*) FILTER (WHERE rate > 0) AS good_updown,
	       COUNT(*) FILTER (WHERE rate >= 3.0) AS good_stars
	FROM web_vote
	GROUP BY article_id
) v ON v.article_id = a.id`

const authorJoin = `
LEFT JOIN (
	SELECT link.article_id, u.username
	FROM web_article_authors link
	JOIN web_user u ON u.id = link.user_id
) au ON au.article_id = a.id`

func (f ListFilter) build(b *listBuilder) string {
	if len(f.Hidden) > 0 {
		b.where = append(b.where, "NOT (a.category = ANY("+b.arg(f.Hidden)+"))")
	}
	switch f.PageType {
	case PageTypeNormal:
		b.where = append(b.where, `a.name NOT LIKE '\_%'`)
	case PageTypeHidden:
		b.where = append(b.where, `a.name LIKE '\_%'`)
	}
	if f.HasName {
		b.where = append(b.where, "a.name = "+b.arg(f.Name))
	}
	if f.HasNamePrefix {
		b.where = append(b.where, "a.name LIKE "+b.arg(likePrefix(f.NamePrefix)))
	}
	if f.NoTags {
		b.where = append(b.where, "NOT EXISTS (SELECT 1 FROM web_article_tags t WHERE t.article_id = a.id)")
	}
	f.buildTags(b)
	if len(f.Categories) > 0 {
		b.where = append(b.where, "a.category = ANY("+b.arg(f.Categories)+")")
	}
	if len(f.NotCategories) > 0 {
		b.where = append(b.where, "NOT (a.category = ANY("+b.arg(f.NotCategories)+"))")
	}
	if f.HasParent {
		if f.ParentID == nil {
			b.where = append(b.where, "a.parent_id IS NULL")
		} else {
			b.where = append(b.where, "a.parent_id = "+b.arg(*f.ParentID))
		}
	}
	if f.HasNotParent {
		if f.NotParentID == nil {
			b.where = append(b.where, "a.parent_id IS NOT NULL")
		} else {
			b.where = append(b.where, "a.parent_id IS DISTINCT FROM "+b.arg(*f.NotParentID))
		}
	}
	if f.AuthorID != nil {
		b.where = append(b.where, "EXISTS (SELECT 1 FROM web_article_authors au"+
			" WHERE au.article_id = a.id AND au.user_id = "+b.arg(*f.AuthorID)+")")
	}
	f.buildCreatedAt(b)
	f.buildNumbers(b)

	sql := "FROM web_article a" + voteJoin
	// Sorting by author multiplies the row rather than picking one of them, so
	// a page with two authors is listed twice.
	if f.Sort.Column == SortCreatedBy {
		sql += authorJoin
	}
	if len(b.where) > 0 {
		sql += "\nWHERE " + strings.Join(b.where, "\n  AND ")
	}
	return sql
}

func (f ListFilter) buildTags(b *listBuilder) {
	hasAll := func(ids []int64) {
		if len(ids) == 0 {
			return
		}
		b.where = append(b.where, "(SELECT COUNT(DISTINCT t.tag_id) FROM web_article_tags t"+
			" WHERE t.article_id = a.id AND t.tag_id = ANY("+b.arg(ids)+")) = "+b.arg(len(ids)))
	}
	hasAll(f.ExactTags)
	hasAll(f.RequiredTags)
	if len(f.PresentTags) > 0 {
		b.where = append(b.where, "EXISTS (SELECT 1 FROM web_article_tags t"+
			" WHERE t.article_id = a.id AND t.tag_id = ANY("+b.arg(f.PresentTags)+"))")
	}
	if len(f.AbsentTags) > 0 {
		b.where = append(b.where, "NOT EXISTS (SELECT 1 FROM web_article_tags t"+
			" WHERE t.article_id = a.id AND t.tag_id = ANY("+b.arg(f.AbsentTags)+"))")
	}
}

func (f ListFilter) buildCreatedAt(b *listBuilder) {
	c := f.CreatedAt
	if c == nil {
		return
	}
	switch c.Op {
	case TimeRange:
		b.where = append(b.where, "a.created_at >= "+b.arg(c.Start)+" AND a.created_at <= "+b.arg(c.End))
	case TimeExcludeRange:
		b.where = append(b.where, "(a.created_at < "+b.arg(c.Start)+" OR a.created_at > "+b.arg(c.End)+")")
	case TimeLT:
		b.where = append(b.where, "a.created_at < "+b.arg(c.Start))
	case TimeLTE:
		b.where = append(b.where, "a.created_at <= "+b.arg(c.Start))
	case TimeGT:
		b.where = append(b.where, "a.created_at > "+b.arg(c.End))
	case TimeGTE:
		b.where = append(b.where, "a.created_at >= "+b.arg(c.End))
	}
}

func (f ListFilter) buildNumbers(b *listBuilder) {
	add := func(expr string, n *NumFilter) {
		if n == nil {
			return
		}
		arg := b.arg(n.Value)
		switch n.Op {
		case NumEQ:
			b.where = append(b.where, expr+" = "+arg)
		case NumNE:
			b.where = append(b.where, "NOT ("+expr+" = "+arg+")")
		case NumLT:
			b.where = append(b.where, expr+" < "+arg)
		case NumLTE:
			b.where = append(b.where, expr+" <= "+arg)
		case NumGT:
			b.where = append(b.where, expr+" > "+arg)
		case NumGTE:
			b.where = append(b.where, expr+" >= "+arg)
		}
	}
	add(ratingExpr(f.RatingMode), f.Rating)
	add(votesExpr, f.Votes)
	add(popularityExpr(f.RatingMode), f.Popularity)
}

func likePrefix(prefix string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(prefix) + "%"
}

// orderBy puts the sort column in the select list too, which SELECT DISTINCT
// requires of anything it orders by.
func (f ListFilter) orderBy() (expr, extra string) {
	direction := " ASC"
	if !f.Sort.Ascending {
		direction = " DESC"
	}
	switch f.Sort.Column {
	case SortCreatedBy:
		return "author_name" + direction, ", au.username AS author_name"
	case SortName:
		return "a.name" + direction, ""
	case SortTitle:
		return "a.title" + direction, ""
	case SortUpdatedAt:
		return "a.updated_at" + direction, ""
	case SortFullName:
		return "full_name" + direction, ", a.complete_full_name AS full_name"
	case SortRating:
		return "rating" + direction, ", " + ratingExpr(f.RatingMode) + " AS rating"
	case SortVotes:
		return "num_votes" + direction, ", " + votesExpr + " AS num_votes"
	case SortPopularity:
		return "popularity" + direction, ", " + popularityExpr(f.RatingMode) + " AS popularity"
	case SortRandom:
		return "shuffled", ", RANDOM() AS shuffled"
	}
	// A column nobody recognises falls back to the newest first, direction and
	// all, so asking for one is not the same as asking for created_at.
	return "a.created_at DESC", ""
}

func (f ListFilter) selectSQL(b *listBuilder, offset int, limit *int) string {
	body := f.build(b)
	order, extra := f.orderBy()
	sql := "SELECT DISTINCT " + prefixedArticleColumns + extra + "\n" + body + "\nORDER BY " + order
	if limit != nil {
		sql += "\nLIMIT " + b.arg(*limit)
	}
	if offset > 0 {
		sql += "\nOFFSET " + b.arg(offset)
	}
	return sql
}

// Exposed so the schema-drift test can send a built statement to Postgres the
// way it sends the ones written out by hand.
func (f ListFilter) SelectSQL(offset int, limit *int) (string, []any) {
	b := &listBuilder{}
	return f.selectSQL(b, offset, limit), b.args
}

func (d *DB) ListArticles(ctx context.Context, f ListFilter, offset int, limit *int) ([]Article, error) {
	b := &listBuilder{}
	sql := f.selectSQL(b, offset, limit)

	rows, err := d.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("query listed articles: %w", err)
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		dest := []any{&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID,
			&a.Locked, &a.CreatedAt, &a.UpdatedAt, &a.MediaName}
		for i := len(dest); i < len(rows.FieldDescriptions()); i++ {
			dest = append(dest, new(any))
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan listed article: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read listed articles: %w", err)
	}
	return out, nil
}

// Counted before pagination narrows it, so offset and limit still apply.
func (d *DB) CountArticles(ctx context.Context, f ListFilter, offset int, limit *int) (int, error) {
	b := &listBuilder{}
	inner := f.selectSQL(b, offset, limit)

	var n int
	if err := d.pool.QueryRow(ctx, "SELECT COUNT(*) FROM ("+inner+") sub", b.args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count listed articles: %w", err)
	}
	return n, nil
}
