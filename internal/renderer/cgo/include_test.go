//go:build cgo && !nocgo

package cgo

import (
	"context"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type pagesHost struct {
	*stubHost
	pages map[string]string
}

func (h *pagesHost) IncludePages(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		page := renderer.FetchedPage{FullName: ref.FullName}
		if body, ok := h.pages[ref.FullName]; ok {
			page.Content = &body
		}
		out = append(out, page)
	}
	return out, nil
}

func TestRenderHTMLIncludeFallback(t *testing.T) {
	host := &pagesHost{stubHost: newHost(), pages: map[string]string{
		"component:block":      "[[include component:block-base width={$width}|width=300px|align={$align}|align=right]]",
		"component:block-base": `[[div class="block-{$align}" style="width:{$width};"]]x[[/div]]`,
	}}
	cases := []struct{ source, want string }{
		{"[[include component:block]]", `<div class="block-right" style="width:300px;">`},
		{"[[include component:block width=200px]]", `<div class="block-right" style="width:200px;">`},
		{"[[include component:block align=left]]", `<div class="block-left" style="width:300px;">`},
	}
	for _, c := range cases {
		got, err := New().RenderHTML(context.Background(), c.source, info(), host, renderer.ModeArticle)
		if err != nil {
			t.Fatalf("RenderHTML(%q) err = %v, want nil", c.source, err)
		}
		if !strings.Contains(got.Body, c.want) {
			t.Errorf("RenderHTML(%q) = %q, want substring %q", c.source, got.Body, c.want)
		}
	}
}

func TestRenderHTMLMissingIncludeVariableIsEmpty(t *testing.T) {
	host := &pagesHost{stubHost: newHost(), pages: map[string]string{
		"component:plain": `[[div class="a-{$class}"]]x[[/div]]`,
	}}
	source := "[[include component:plain]]"
	got, err := New().RenderHTML(context.Background(), source, info(), host, renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML(%q) err = %v, want nil", source, err)
	}
	if want := `<div class="a-">`; !strings.Contains(got.Body, want) {
		t.Errorf("RenderHTML(%q) = %q, want substring %q", source, got.Body, want)
	}
}
