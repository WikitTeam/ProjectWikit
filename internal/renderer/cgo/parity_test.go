//go:build cgo && !nocgo

package cgo

import (
	"context"
	"os"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

func TestParityWithSidecar(t *testing.T) {
	binary := os.Getenv(sidecar.EnvBinary)
	if binary == "" {
		t.Skipf("%s not set, skipping the cgo/sidecar parity test", sidecar.EnvBinary)
	}

	sources := []string{
		"//italic// and **bold**",
		"[[[exists|blue]]] and [[[missing|red]]]",
		`[[module Rate limit="5"]]`,
		"[[module ListPages]]\nbody\n[[/module]]",
		"[[include exists |a=1 |b=2]]",
		"[[include no-such-page]]",
		"[[user someone]] and [[*user other]]",
		"[[#expr 1 + 1]]",
		"text[[footnote]]note[[/footnote]]",
		"[[toc]]\n\n+ heading",
		"[[code type=\"python\"]]\nprint(1)\n[[/code]]",
		"> quote\n\n* item\n* item",
	}

	linked := New()
	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			piped, err := sidecar.New(binary)
			if err != nil {
				t.Fatalf("sidecar.New(%q) err = %v, want nil", binary, err)
			}
			defer piped.Close()

			viaSidecar, err := piped.RenderHTML(context.Background(), source, info(), newHost(), renderer.ModeArticle)
			if err != nil {
				t.Fatalf("sidecar RenderHTML(%q) err = %v, want nil", source, err)
			}
			viaCgo, err := linked.RenderHTML(context.Background(), source, info(), newHost(), renderer.ModeArticle)
			if err != nil {
				t.Fatalf("cgo RenderHTML(%q) err = %v, want nil", source, err)
			}

			if viaSidecar.Body == "" {
				t.Fatalf("sidecar body = \"\", want rendered output")
			}
			if viaCgo.Body != viaSidecar.Body {
				t.Errorf("cgo body = %q, want %q", viaCgo.Body, viaSidecar.Body)
			}
			if len(viaCgo.LinkedPages) != len(viaSidecar.LinkedPages) {
				t.Errorf("len(cgo LinkedPages) = %d, want %d", len(viaCgo.LinkedPages), len(viaSidecar.LinkedPages))
			}
			if len(viaCgo.IncludedPages) != len(viaSidecar.IncludedPages) {
				t.Errorf("len(cgo IncludedPages) = %d, want %d", len(viaCgo.IncludedPages), len(viaSidecar.IncludedPages))
			}
			if len(viaCgo.Code) != len(viaSidecar.Code) {
				t.Errorf("len(cgo Code) = %d, want %d", len(viaCgo.Code), len(viaSidecar.Code))
			}
		})
	}
}

func TestCallbackTraceParity(t *testing.T) {
	binary := os.Getenv(sidecar.EnvBinary)
	if binary == "" {
		t.Skipf("%s not set, skipping the cgo/sidecar parity test", sidecar.EnvBinary)
	}

	source := "[[[exists|a]]] [[[missing|b]]]\n\n[[include exists |a=1]]\n\n[[module Rate limit=\"5\"]]"

	piped, err := sidecar.New(binary)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v, want nil", binary, err)
	}
	defer piped.Close()

	sidecarHost := newHost()
	if _, err := piped.RenderHTML(context.Background(), source, info(), sidecarHost, renderer.ModeArticle); err != nil {
		t.Fatalf("sidecar RenderHTML() err = %v, want nil", err)
	}

	cgoHost := newHost()
	if _, err := New().RenderHTML(context.Background(), source, info(), cgoHost, renderer.ModeArticle); err != nil {
		t.Fatalf("cgo RenderHTML() err = %v, want nil", err)
	}

	if len(sidecarHost.calls) == 0 {
		t.Fatal("sidecar calls = [], want the callbacks this source triggers")
	}
	if len(cgoHost.calls) != len(sidecarHost.calls) {
		t.Fatalf("cgo calls = %v, want %v", cgoHost.calls, sidecarHost.calls)
	}
	for i := range sidecarHost.calls {
		if cgoHost.calls[i] != sidecarHost.calls[i] {
			t.Errorf("cgo calls[%d] = %q, want %q", i, cgoHost.calls[i], sidecarHost.calls[i])
		}
	}
}
