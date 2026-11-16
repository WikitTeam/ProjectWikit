package listpages

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func TestParseRangeBeforeAndAfter(t *testing.T) {
	for _, r := range []string{RangeBefore, RangeAfter} {
		q := parse(t, newFakeSource(), article173(), nil, map[string]string{"range": r})
		if q.Range != r || q.Around != 7 {
			t.Errorf("Parse(range=%s) Range, Around = %q, %d, want %q, 7", r, q.Range, q.Around, r)
		}
	}
	if q := parse(t, newFakeSource(), nil, nil, map[string]string{"range": RangeBefore}); !q.Invalid {
		t.Errorf("Parse(range=before) without a page Invalid = false, want true")
	}
}

func TestRunRangeCutsAroundThePage(t *testing.T) {
	src := newFakeSource()
	src.listed = []db.Article{{ID: 1}, {ID: 7}, {ID: 9}, {ID: 12}}
	cases := map[string][]int64{RangeBefore: {1}, RangeAfter: {9, 12}}
	for r, want := range cases {
		q := parse(t, src, article173(), nil, map[string]string{"range": r, "category": "*"})
		res, err := Run(src, q, nil, false)
		if err != nil {
			t.Fatalf("Run(range=%s) err = %v, want nil", r, err)
		}
		var got []int64
		for _, a := range res.Pages {
			got = append(got, a.ID)
		}
		if len(got) != len(want) {
			t.Errorf("Run(range=%s) ids = %v, want %v", r, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("Run(range=%s) ids = %v, want %v", r, got, want)
				break
			}
		}
	}
}

func TestParseCreatedByTheAuthorOfThePage(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_by": "="})
	if q.Filter.SameAuthorsAs == nil || *q.Filter.SameAuthorsAs != 7 {
		t.Errorf("Parse(created_by==) SameAuthorsAs = %v, want 7", q.Filter.SameAuthorsAs)
	}
	q = parse(t, newFakeSource(), article173(), nil, map[string]string{"created_by": "-="})
	if q.Filter.NotSameAuthorsAs == nil || *q.Filter.NotSameAuthorsAs != 7 {
		t.Errorf("Parse(created_by=-=) NotSameAuthorsAs = %v, want 7", q.Filter.NotSameAuthorsAs)
	}
}

func TestParseCreatedAtWithDots(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_at": "2008.07"})
	f := q.Filter.CreatedAt
	if f == nil {
		t.Fatalf("Parse(created_at=2008.07) CreatedAt = nil, want a range")
	}
	if want := time.Date(2008, 7, 1, 0, 0, 0, 0, time.UTC); !f.Start.Equal(want) {
		t.Errorf("Parse(created_at=2008.07) Start = %v, want %v", f.Start, want)
	}
	if want := time.Date(2008, 8, 1, 0, 0, 0, 0, time.UTC); !f.End.Equal(want) {
		t.Errorf("Parse(created_at=2008.07) End = %v, want %v", f.End, want)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in    string
		op    string
		start time.Time
	}{
		{"last 3 day", db.TimeGTE, now.AddDate(0, 0, -3)},
		{"last week", db.TimeGTE, now.AddDate(0, 0, -7)},
		{"Last 2 Hours", db.TimeGTE, now.Add(-2 * time.Hour)},
		{"older than 1 month", db.TimeLT, now.AddDate(0, -1, 0)},
	}
	for _, c := range cases {
		got, ok := relativeTime(c.in, now)
		if !ok {
			t.Errorf("relativeTime(%q) ok = false, want true", c.in)
			continue
		}
		if got.Op != c.op || !got.Start.Equal(c.start) {
			t.Errorf("relativeTime(%q) = %s %v, want %s %v", c.in, got.Op, got.Start, c.op, c.start)
		}
	}
	if _, ok := relativeTime("2008.07", now); ok {
		t.Errorf("relativeTime(%q) ok = true, want false", "2008.07")
	}
}

func TestParseOrderDropsTheCast(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"order": "_albums::integer desc"})
	if q.FormSort != "albums" || q.FormSortAsc {
		t.Errorf("Parse(order=_albums::integer desc) FormSort, asc = %q, %v, want %q, false", q.FormSort, q.FormSortAsc, "albums")
	}
}

func TestParseCategoryVariable(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"category": "%%category%%"})
	if len(q.Filter.Categories) != 1 || q.Filter.Categories[0] != "scp" {
		t.Errorf("Parse(category=%%%%category%%%%) Categories = %v, want [scp]", q.Filter.Categories)
	}
}

func TestURLAttrPrefix(t *testing.T) {
	path := page.ParsePathParams(`{"a_created_at": "2008", "a_p": "3", "p": "9"}`)
	params, _ := URLParams(map[string]string{"created_at": "@URL", "urlattrprefix": "a_"}, path)
	if params["created_at"] != "2008" {
		t.Errorf("URLParams() created_at = %q, want %q", params["created_at"], "2008")
	}
	if key := PageKey(params); key != "a_p" {
		t.Errorf("PageKey() = %q, want %q", key, "a_p")
	}
	q, err := Parse(newFakeSource(), article173(), nil, nil, params, path)
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	if q.Page != 3 {
		t.Errorf("Parse() Page = %d, want 3", q.Page)
	}
	if got := KeyedBasePath("main", path, "a_p"); got != "/main/a_created_at/2008/p/9" {
		t.Errorf("KeyedBasePath() = %q, want %q", got, "/main/a_created_at/2008/p/9")
	}
}
