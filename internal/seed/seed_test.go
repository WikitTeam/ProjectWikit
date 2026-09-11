package seed

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type fakeStore struct {
	existing map[string]bool
	created  []string
	versions []string
	nextID   int64
}

func (f *fakeStore) ArticleByName(_ context.Context, _ int64, ref string) (*db.Article, error) {
	if f.existing[ref] {
		return &db.Article{}, nil
	}
	return nil, db.ErrNotFound
}

func (f *fakeStore) CreateArticle(_ context.Context, _ int64, category, name, _ string, _ *int64, _ time.Time) (int64, error) {
	f.nextID++
	f.created = append(f.created, category+":"+name)
	return f.nextID, nil
}

func (f *fakeStore) CreateArticleVersion(_ context.Context, w db.VersionWrite) (db.Revision, error) {
	if w.Source == "" {
		return db.Revision{}, nil
	}
	f.versions = append(f.versions, w.Kind)
	return db.Revision{}, nil
}

func TestNamesTurnDirectoriesIntoCategories(t *testing.T) {
	got := Names()
	for _, want := range []string{"main", "nav:top", "nav:side", "forum:start", "search:site"} {
		if !slices.Contains(got, want) {
			t.Errorf("Names() = %v, want it to contain %q", got, want)
		}
	}
	for _, unwanted := range []string{"nav/top", "main.ftml", "pages/main"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("Names() contains %q, want it absent", unwanted)
		}
	}
}

func TestRunWritesEveryPageOnce(t *testing.T) {
	store := &fakeStore{existing: map[string]bool{}}
	written, err := Run(context.Background(), store, 1)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if len(written) != len(Names()) {
		t.Errorf("len(Run()) = %d, want %d", len(written), len(Names()))
	}
	for _, kind := range store.versions {
		if kind != db.LogNew {
			t.Errorf("version kind = %q, want %q", kind, db.LogNew)
		}
	}
}

func TestRunSkipsPagesThatAreAlreadyThere(t *testing.T) {
	store := &fakeStore{existing: map[string]bool{"main": true, "nav:top": true}}
	written, err := Run(context.Background(), store, 1)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if slices.Contains(written, "main") {
		t.Errorf("Run() = %v, want it to leave main alone", written)
	}
	if len(written) != len(Names())-2 {
		t.Errorf("len(Run()) = %d, want %d", len(written), len(Names())-2)
	}
}

func TestSplitPutsUncategorisedPagesInTheDefaultCategory(t *testing.T) {
	cases := map[string][2]string{
		"main":             {"_default", "main"},
		"nav:top":          {"nav", "top"},
		"forum:new-thread": {"forum", "new-thread"},
	}
	for full, want := range cases {
		category, name := split(full)
		if category != want[0] || name != want[1] {
			t.Errorf("split(%q) = %q, %q, want %q, %q", full, category, name, want[0], want[1])
		}
	}
}
