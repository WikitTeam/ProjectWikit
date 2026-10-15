//go:build cgo && !nocgo

package cgo

import (
	"context"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func TestRenderHTMLConditionalsInHeads(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    string
		without string
	}{
		{"if in an attribute", `[[span class="[[#if 1 | on | off ]]"]]x[[/span]]`, `class="on"`, "#if"},
		{"zero is false", `[[span class="[[#if 0 | on | off ]]"]]x[[/span]]`, `class="off"`, "#if"},
		{"empty is false", `[[span class="[[#if  | on | off ]]"]]x[[/span]]`, `class="off"`, "#if"},
		{"whole attribute", "[[div [[#if 0 | | class=\"end\" ]] ]]\nx\n[[/div]]", `<div class="end">`, "#if"},
		{"ifexpr in an attribute", `[[span class="[[#ifexpr 2 > 1 | big | small ]]"]]x[[/span]]`, `class="big"`, "#ifexpr"},
		{"raw text kept", `@@[[span class="[[#if 1 | on | off ]]"]]@@`, "#if 1", `class="on"`},
		{"branch trimmed", "a[[#if 0 | | b ]]c", "abc", "a b"},
		{"zero in the body", "[[#if 0 | yes | no ]]", "no", "yes"},
	}

	r := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.RenderHTML(context.Background(), tt.source, info(), newHost(), renderer.ModeArticle)
			if err != nil {
				t.Fatalf("RenderHTML(%q) err = %v, want nil", tt.source, err)
			}
			if !strings.Contains(got.Body, tt.want) {
				t.Errorf("RenderHTML(%q) = %q, want substring %q", tt.source, got.Body, tt.want)
			}
			if strings.Contains(got.Body, tt.without) {
				t.Errorf("RenderHTML(%q) = %q, want no %q", tt.source, got.Body, tt.without)
			}
		})
	}
}
