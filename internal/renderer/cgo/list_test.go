//go:build cgo && !nocgo

package cgo

import (
	"context"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func TestRenderHTMLListWrapsLooseContent(t *testing.T) {
	source := "[[ul]][[li]]a[[/li]][[ul]]_[[/ul]][[/ul]]"
	got, err := New().RenderHTML(context.Background(), source, info(), newHost(), renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML(%q) err = %v, want nil", source, err)
	}
	if want := `<li style="list-style: none">_</li>`; !strings.Contains(got.Body, want) {
		t.Errorf("RenderHTML(%q) = %q, want substring %q", source, got.Body, want)
	}
	if strings.Contains(got.Body, "[[ul]]") {
		t.Errorf("RenderHTML(%q) = %q, want no %q", source, got.Body, "[[ul]]")
	}
}
