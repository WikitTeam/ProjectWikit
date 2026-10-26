//go:build cgo && !nocgo

package cgo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type includeRepo struct {
	pages map[string]string
}

func (r *includeRepo) RenderModule(*page.Context, string, map[string]string, string) (string, error) {
	return "", nil
}

func (r *includeRepo) RenderUser(string, bool) (string, error) { return "", nil }

func (r *includeRepo) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	out := make([]renderer.PartialPageInfo, 0, len(refs))
	for _, ref := range refs {
		out = append(out, renderer.PartialPageInfo{FullName: ref})
	}
	return out, nil
}

func (r *includeRepo) IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		fetched := renderer.FetchedPage{FullName: ref.FullName}
		if body, ok := r.pages[ref.FullName]; ok {
			fetched.Content = &body
		}
		out = append(out, fetched)
	}
	return out, nil
}

func TestRenderHTMLStopsIncludesThatMultiply(t *testing.T) {
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	loc := bundle.Localizer(i18n.DefaultLanguage)
	filler := strings.Repeat("x", 20000) + "\n\n"
	repo := &includeRepo{pages: map[string]string{
		"component:animation":    filler + "[[include component:animation-bk]]\n[[include component:animation]]\n[[include component:animation]]",
		"component:animation-bk": filler + "[[include component:animation]]",
	}}
	cb := callbacks.New(loc, repo)

	start := time.Now()
	got, err := New().RenderHTML(context.Background(), "[[include component:animation]]", info(), cb, renderer.ModeArticle)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("RenderHTML() err = %v, want nil", err)
	}
	t.Logf("RenderHTML() took %v for %d bytes", elapsed, len(got.Body))
	stop := strings.SplitN(loc.T("include-too-large", "name", "x"), `"`, 2)[0]
	if !strings.Contains(got.Body, stop) {
		t.Errorf("RenderHTML() body lacks %q", stop)
	}
}
