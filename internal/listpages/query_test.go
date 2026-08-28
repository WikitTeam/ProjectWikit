package listpages

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func intPtr(n int) *int { return &n }

func TestWindowStacksPaginationOnTheOffset(t *testing.T) {
	offset, limit, skip := window(Query{Offset: 5, Page: 3, PerPage: 20})
	if skip {
		t.Fatal("window().skip = true, want false")
	}
	if offset != 45 {
		t.Errorf("window().offset = %d, want 45", offset)
	}
	if limit == nil || *limit != 20 {
		t.Errorf("window().limit = %v, want 20", limit)
	}
}

func TestWindowNarrowsTheLastPageToTheLimit(t *testing.T) {
	offset, limit, skip := window(Query{Offset: 0, Limit: intPtr(25), Page: 2, PerPage: 20})
	if skip {
		t.Fatal("window().skip = true, want false")
	}
	if offset != 20 {
		t.Errorf("window().offset = %d, want 20", offset)
	}
	if limit == nil || *limit != 5 {
		t.Errorf("window().limit = %v, want 5", limit)
	}
}

func TestWindowSkipsAPageBeyondTheLimit(t *testing.T) {
	if _, _, skip := window(Query{Limit: intPtr(10), Page: 2, PerPage: 20}); !skip {
		t.Error("window().skip = false, want true")
	}
}

func TestWindowSkipsWhenNoPageHasRoom(t *testing.T) {
	if _, _, skip := window(Query{Page: 1, PerPage: 0}); !skip {
		t.Error("window(perpage=0).skip = false, want true")
	}
}

func TestTotalPagesRoundsUp(t *testing.T) {
	cases := []struct{ total, perPage, want int }{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{40, 20, 2},
		{5, 0, 0},
	}
	for _, c := range cases {
		if got := totalPages(c.total, c.perPage); got != c.want {
			t.Errorf("totalPages(%d, %d) = %d, want %d", c.total, c.perPage, got, c.want)
		}
	}
}

func TestRunCapsThePageSize(t *testing.T) {
	src := newFakeSource()
	src.total = 1000
	got, err := Run(src, Query{Page: 1, PerPage: 300}, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if src.gotLimit == nil || *src.gotLimit != 250 {
		t.Errorf("limit = %v, want 250", src.gotLimit)
	}
	if got.TotalPages != 4 {
		t.Errorf("TotalPages = %d, want 4", got.TotalPages)
	}
}

func TestRunOfAnInvalidQueryListsNothing(t *testing.T) {
	src := newFakeSource()
	got, err := Run(src, Query{Invalid: true}, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if len(got.Pages) != 0 || got.Total != 0 || got.Page != 1 || got.TotalPages != 1 {
		t.Errorf("Run(invalid) = %+v, want an empty first page", got)
	}
	if src.listCalls != 0 {
		t.Errorf("listCalls = %d, want 0", src.listCalls)
	}
}

func TestRunOfASinglePage(t *testing.T) {
	src := newFakeSource()
	only := article173()
	got, err := Run(src, Query{HasOnly: true, Only: only}, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if len(got.Pages) != 1 || got.Pages[0].ID != 7 || got.Total != 1 {
		t.Errorf("Run(only) = %+v, want the one article", got)
	}
}

func TestRunOfASinglePageInAHiddenCategory(t *testing.T) {
	src := newFakeSource()
	src.hidden = []string{"SCP"}
	got, err := Run(src, Query{HasOnly: true, Only: article173()}, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if len(got.Pages) != 0 {
		t.Errorf("Run(only, hidden) = %+v, want nothing", got)
	}
}

func TestRunOfAFullNameThatDoesNotExist(t *testing.T) {
	src := newFakeSource()
	got, err := Run(src, Query{HasFullName: true, FullName: "no-such-page"}, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if len(got.Pages) != 0 || got.Total != 0 {
		t.Errorf("Run(fullname) = %+v, want nothing", got)
	}
}

func TestRunPassesTheHiddenCategoriesToTheFilter(t *testing.T) {
	src := newFakeSource()
	src.hidden = []string{"admin"}
	src.total = 3
	if _, err := Run(src, Query{Page: 1, PerPage: 20}, nil, true); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if len(src.gotFilter.Hidden) != 1 || src.gotFilter.Hidden[0] != "admin" {
		t.Errorf("Hidden = %v, want [admin]", src.gotFilter.Hidden)
	}
}

func TestRunReportsThePageIndexOfTheFirstRow(t *testing.T) {
	src := newFakeSource()
	src.total = 45
	src.listed = []db.Article{*article173()}
	got, err := Run(src, Query{Page: 3, PerPage: 20}, nil, true)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if got.PageIndex != 40 {
		t.Errorf("PageIndex = %d, want 40", got.PageIndex)
	}
	if got.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", got.TotalPages)
	}
}

func TestRunWithoutPaginationIgnoresThePageNumber(t *testing.T) {
	src := newFakeSource()
	src.total = 45
	got, err := Run(src, Query{Offset: 2, Page: 3, PerPage: 20}, nil, false)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if src.gotOffset != 2 {
		t.Errorf("offset = %d, want 2", src.gotOffset)
	}
	if got.TotalPages != 1 || got.Page != 1 {
		t.Errorf("Run(no pagination) page, total = %d, %d, want 1, 1", got.Page, got.TotalPages)
	}
}
