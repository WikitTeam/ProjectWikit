//go:build cgo && !nocgo

package cgo

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func TestRenderHTMLSandboxNeedsAClickToLeave(t *testing.T) {
	source := "[[html]]\n<p>x</p>\n[[/html]]"
	got, err := New().RenderHTML(context.Background(), source, info(), newHost(), renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML(%q) err = %v, want nil", source, err)
	}
	_, rest, ok := strings.Cut(got.Body, `sandbox="`)
	if !ok {
		t.Fatalf("RenderHTML(%q) = %q, want a sandbox attribute", source, got.Body)
	}
	tokens, _, _ := strings.Cut(rest, `"`)
	for _, token := range strings.Fields(tokens) {
		if token == "allow-top-navigation" {
			t.Errorf("RenderHTML(%q) sandbox = %q, want no %q", source, tokens, token)
		}
	}
	if !slices.Contains(strings.Fields(tokens), "allow-top-navigation-by-user-activation") {
		t.Errorf("RenderHTML(%q) sandbox = %q, want %q", source, tokens, "allow-top-navigation-by-user-activation")
	}
}
