package listpages

import (
	"sort"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
)

func (q Query) hasFormWork() bool { return len(q.FormConds) > 0 || q.FormSort != "" }

// Form fields live in the page source rather than in a column, so the rows the
// database can narrow are read one by one and the rest of the query is
// answered here.
func runWithForm(src Source, q Query, filter db.ListFilter, paginate bool) (Result, error) {
	rows, err := src.ListArticles(filter, 0, nil)
	if err != nil {
		return Result{}, err
	}
	kept, err := selectByForm(src, q, rows)
	if err != nil {
		return Result{}, err
	}

	kept = kept[min(q.Offset, len(kept)):]
	if q.Limit != nil && *q.Limit < len(kept) {
		kept = kept[:max(*q.Limit, 0)]
	}
	total := len(kept)

	if !paginate {
		return Result{Pages: kept, Page: 1, TotalPages: 1, Total: total}, nil
	}
	if q.PerPage <= 0 {
		return Result{Page: q.Page, TotalPages: 0, Total: total}, nil
	}
	start := min((q.Page-1)*q.PerPage, total)
	end := min(start+q.PerPage, total)
	return Result{
		Pages:      kept[start:end],
		PageIndex:  (q.Page - 1) * q.PerPage,
		Page:       q.Page,
		TotalPages: totalPages(total, q.PerPage),
		Total:      total,
	}, nil
}

func selectByForm(src Source, q Query, rows []db.Article) ([]db.Article, error) {
	kept := make([]db.Article, 0, len(rows))
	keys := make(map[int64]string, len(rows))

	for _, row := range rows {
		values, def, err := formValuesOf(src, row)
		if err != nil {
			return nil, err
		}
		if !matchesAll(def, values, q.FormConds) {
			continue
		}
		if q.FormSort != "" {
			key, _ := def.Raw(values, q.FormSort)
			keys[row.ID] = key
		}
		kept = append(kept, row)
	}

	if q.FormSort != "" {
		// Stable, so rows carrying the same value keep the order the database
		// put them in.
		sort.SliceStable(kept, func(i, j int) bool {
			left, right := keys[kept[i].ID], keys[kept[j].ID]
			if q.FormSortAsc {
				return left < right
			}
			return left > right
		})
	}
	return kept, nil
}

func formValuesOf(src Source, row db.Article) (map[string]string, *form.Definition, error) {
	def, err := src.CategoryForm(row.Category)
	if err != nil {
		return nil, nil, err
	}
	if def == nil {
		return nil, nil, nil
	}
	source, err := src.LatestSource(row.ID)
	if err != nil {
		// A page with no revision answers with whatever the form defaults to.
		source = ""
	}
	values, err := form.ParseData(source)
	if err != nil {
		values = map[string]string{}
	}
	return values, def, nil
}

// A row whose category carries no form has no value to compare, so a condition
// on one excludes it rather than matching an empty string.
func matchesAll(def *form.Definition, values map[string]string, conds []FormCond) bool {
	for _, cond := range conds {
		if def == nil {
			return false
		}
		value, ok := def.Raw(values, cond.Field)
		if !ok {
			return false
		}
		if !compareForm(value, cond.Op, cond.Value) {
			return false
		}
	}
	return true
}

func compareForm(value, op, want string) bool {
	switch op {
	case "<>":
		return value != want
	case "<":
		return value < want
	case ">":
		return value > want
	case "<=":
		return value <= want
	case ">=":
		return value >= want
	}
	return strings.EqualFold(value, want)
}
