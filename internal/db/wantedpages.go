package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type WantedLink struct {
	From  string
	To    string
	Title string
}

type WantedFilter struct {
	From          []string
	Categories    []string
	NotCategories []string
}

// A link recorded without a category belongs to the default one, on both ends,
// and the stored text carries no hint of that.
const (
	completeFrom = `(CASE WHEN position(':' in l.link_from) > 0 THEN l.link_from::text ELSE '_default:' || l.link_from::text END)`
	completeTo   = `(CASE WHEN position(':' in l.link_to) > 0 THEN l.link_to::text ELSE '_default:' || l.link_to::text END)`
	categoryOfTo = `(CASE WHEN position(':' in l.link_to) > 0 THEN substring(l.link_to::text from 1 for position(':' in l.link_to) - 1) ELSE '_default' END)`
)

func (f WantedFilter) where(args *[]any) string {
	*args = append(*args, lowerAll(f.From))
	var b strings.Builder
	b.WriteString("WHERE l.link_type = 'link'\n  AND lower(" + completeFrom + ") = ANY($1)" +
		"\n  AND NOT EXISTS (SELECT 1 FROM web_article a WHERE a.complete_full_name = " + completeTo + ")")
	if len(f.Categories) > 0 {
		*args = append(*args, lowerAll(f.Categories))
		b.WriteString("\n  AND lower(" + categoryOfTo + ") = ANY($" + strconv.Itoa(len(*args)) + ")")
	}
	if len(f.NotCategories) > 0 {
		*args = append(*args, lowerAll(f.NotCategories))
		b.WriteString("\n  AND NOT (lower(" + categoryOfTo + ") = ANY($" + strconv.Itoa(len(*args)) + "))")
	}
	return b.String()
}

// The link table has no order of its own, so the row id is what keeps a page of
// results from reshuffling between requests.
func (d *DB) WantedLinks(ctx context.Context, f WantedFilter, offset, limit int) ([]WantedLink, error) {
	args := []any{}
	where := f.where(&args)
	args = append(args, limit, offset)
	sql := `SELECT l.link_from, l.link_to, coalesce(src.title, '')
FROM web_externallink l
LEFT JOIN web_article src ON src.complete_full_name = ` + completeFrom + "\n" + where +
		"\nORDER BY l.id" +
		"\nLIMIT $" + strconv.Itoa(len(args)-1) + " OFFSET $" + strconv.Itoa(len(args))

	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query wanted links: %w", err)
	}
	defer rows.Close()

	var out []WantedLink
	for rows.Next() {
		var link WantedLink
		if err := rows.Scan(&link.From, &link.To, &link.Title); err != nil {
			return nil, fmt.Errorf("scan wanted link: %w", err)
		}
		out = append(out, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read wanted links: %w", err)
	}
	return out, nil
}

func (d *DB) WantedLinkCount(ctx context.Context, f WantedFilter) (int, error) {
	args := []any{}
	sql := "SELECT count(*)\nFROM web_externallink l\n" + f.where(&args)

	var total int
	if err := d.pool.QueryRow(ctx, sql, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count wanted links: %w", err)
	}
	return total, nil
}

func lowerAll(values []string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = strings.ToLower(value)
	}
	return out
}
