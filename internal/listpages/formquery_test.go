package listpages

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
)

const noticeForm = `[[form]]
fields:
 Pinned:
   type: checkbox
   default: 0
 Rank:
   type: text
[[/form]]`

func noticeSource(t *testing.T) *fakeSource {
	t.Helper()
	def, _, err := form.Parse(noticeForm)
	if err != nil {
		t.Fatalf("form.Parse() err = %v, want nil", err)
	}
	return &fakeSource{
		forms: map[string]*form.Definition{"notice": def},
		listed: []db.Article{
			{ID: 1, Category: "notice", Name: "a"},
			{ID: 2, Category: "notice", Name: "b"},
			{ID: 3, Category: "notice", Name: "c"},
		},
		sources: map[int64]string{
			1: "Pinned: '1'\nRank: '02'",
			2: "Pinned: '0'\nRank: '01'",
			3: "Rank: '03'",
		},
	}
}

func namesOf(pages []db.Article) []string {
	out := make([]string, len(pages))
	for i := range pages {
		out[i] = pages[i].Name
	}
	return out
}

func sameNames(got []db.Article, want []string) bool {
	names := namesOf(got)
	if len(names) != len(want) {
		return false
	}
	for i := range want {
		if names[i] != want[i] {
			return false
		}
	}
	return true
}

func TestRunFiltersOnAFormField(t *testing.T) {
	src := noticeSource(t)
	q := Query{
		Page: 1, PerPage: 20,
		FormConds: []FormCond{{Field: "pinned", Op: "=", Value: "1"}},
	}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"a"}) {
		t.Errorf("Run().Pages = %v, want [a]", namesOf(got.Pages))
	}
	if got.Total != 1 {
		t.Errorf("Run().Total = %d, want 1", got.Total)
	}
}

func TestRunCountsAMissingValueAsTheDefault(t *testing.T) {
	src := noticeSource(t)
	q := Query{
		Page: 1, PerPage: 20,
		FormConds: []FormCond{{Field: "Pinned", Op: "=", Value: "0"}},
	}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"b", "c"}) {
		t.Errorf("Run().Pages = %v, want [b c]", namesOf(got.Pages))
	}
}

func TestRunComparesAFormFieldWithAnOperator(t *testing.T) {
	src := noticeSource(t)
	q := Query{
		Page: 1, PerPage: 20,
		FormConds: []FormCond{{Field: "Rank", Op: ">", Value: "01"}},
	}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"a", "c"}) {
		t.Errorf("Run().Pages = %v, want [a c]", namesOf(got.Pages))
	}
}

func TestRunDropsARowWhoseCategoryHasNoForm(t *testing.T) {
	src := noticeSource(t)
	src.listed = append(src.listed, db.Article{ID: 4, Category: "other", Name: "d"})
	q := Query{
		Page: 1, PerPage: 20,
		FormConds: []FormCond{{Field: "Pinned", Op: "=", Value: "0"}},
	}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"b", "c"}) {
		t.Errorf("Run().Pages = %v, want [b c]", namesOf(got.Pages))
	}
}

func TestRunSortsOnAFormField(t *testing.T) {
	src := noticeSource(t)
	q := Query{Page: 1, PerPage: 20, FormSort: "Rank", FormSortAsc: true}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"b", "a", "c"}) {
		t.Errorf("Run().Pages = %v, want [b a c]", namesOf(got.Pages))
	}
}

func TestRunSortsOnAFormFieldDescending(t *testing.T) {
	src := noticeSource(t)
	q := Query{Page: 1, PerPage: 20, FormSort: "Rank"}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"c", "a", "b"}) {
		t.Errorf("Run().Pages = %v, want [c a b]", namesOf(got.Pages))
	}
}

func TestRunPaginatesTheFilteredRows(t *testing.T) {
	src := noticeSource(t)
	q := Query{
		Page: 2, PerPage: 1,
		FormConds: []FormCond{{Field: "Pinned", Op: "=", Value: "0"}},
	}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"c"}) {
		t.Errorf("Run().Pages = %v, want [c]", namesOf(got.Pages))
	}
	if got.TotalPages != 2 {
		t.Errorf("Run().TotalPages = %d, want 2", got.TotalPages)
	}
	if got.PageIndex != 1 {
		t.Errorf("Run().PageIndex = %d, want 1", got.PageIndex)
	}
}

func TestRunAppliesTheLimitAfterFiltering(t *testing.T) {
	src := noticeSource(t)
	limit := 1
	q := Query{
		Page: 1, PerPage: 20, Limit: &limit,
		FormConds: []FormCond{{Field: "Pinned", Op: "=", Value: "0"}},
	}

	got, err := Run(src, q, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !sameNames(got.Pages, []string{"b"}) {
		t.Errorf("Run().Pages = %v, want [b]", namesOf(got.Pages))
	}
	if got.Total != 1 {
		t.Errorf("Run().Total = %d, want 1", got.Total)
	}
}

func TestParseReadsFormConditions(t *testing.T) {
	q := parse(t, noticeSource(t), nil, nil, map[string]string{"category": "notice", "_pinnednotice": "1", "_rank": ">02"})
	want := []FormCond{{Field: "pinnednotice", Op: "=", Value: "1"}, {Field: "rank", Op: ">", Value: "02"}}
	if len(q.FormConds) != len(want) {
		t.Fatalf("FormConds = %+v, want %+v", q.FormConds, want)
	}
	for i := range want {
		if q.FormConds[i] != want[i] {
			t.Errorf("FormConds[%d] = %+v, want %+v", i, q.FormConds[i], want[i])
		}
	}
}

func TestParseReadsAFormSort(t *testing.T) {
	q := parse(t, noticeSource(t), nil, nil, map[string]string{"category": "notice", "order": "_rank desc"})
	if q.FormSort != "rank" {
		t.Errorf("FormSort = %q, want %q", q.FormSort, "rank")
	}
	if q.FormSortAsc {
		t.Error("FormSortAsc = true, want false")
	}
	if q.Filter.Sort.Column != "created_at" {
		t.Errorf("Filter.Sort.Column = %q, want %q", q.Filter.Sort.Column, "created_at")
	}
}

func TestParseLeavesAnOrdinarySortAlone(t *testing.T) {
	q := parse(t, noticeSource(t), nil, nil, map[string]string{"category": "notice", "order": "name"})
	if q.FormSort != "" {
		t.Errorf("FormSort = %q, want %q", q.FormSort, "")
	}
	if q.Filter.Sort.Column != "name" {
		t.Errorf("Filter.Sort.Column = %q, want %q", q.Filter.Sort.Column, "name")
	}
}
