package pagerender

import (
	"context"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func TestBacklinksKeepsIncludesApartFromLinks(t *testing.T) {
	engine := &renderer.Fake{Result: renderer.Result{
		IncludedPages: []string{"Component:Box"},
		LinkedPages:   []string{"SCP-173"},
	}}
	env := Deps{Engine: engine}.Env(context.Background(), nil, nil, nil)

	got, err := env.Backlinks("source", renderer.PageInfo{}, nil)
	if err != nil {
		t.Fatalf("Backlinks() err = %v, want nil", err)
	}
	want := []db.ArticleLink{
		{To: "component:box", Kind: db.LinkInclude},
		{To: "scp-173", Kind: db.LinkPlain},
	}
	if len(got) != len(want) {
		t.Fatalf("len(Backlinks()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Backlinks()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestBacklinksCollectsInSystemMode(t *testing.T) {
	engine := &renderer.Fake{}
	env := Deps{Engine: engine}.Env(context.Background(), nil, nil, nil)

	if _, err := env.Backlinks("source", renderer.PageInfo{}, nil); err != nil {
		t.Fatalf("Backlinks() err = %v, want nil", err)
	}
	if len(engine.Calls) != 1 {
		t.Fatalf("len(engine calls) = %d, want 1", len(engine.Calls))
	}
	if got, want := engine.Calls[0].Op, "CollectBacklinks"; got != want {
		t.Errorf("engine call = %q, want %q", got, want)
	}
	if got, want := engine.Calls[0].Mode, renderer.ModeSystem; got != want {
		t.Errorf("mode = %q, want %q", got, want)
	}
}
