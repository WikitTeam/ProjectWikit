//go:build cgo && !nocgo

package cgo

import (
	"context"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type stubHost struct {
	renderer.NopCallbacks
	existing map[string]bool
	calls    []string
}

func (h *stubHost) ModuleHasBody(name string) (bool, error) {
	h.calls = append(h.calls, "module_has_body:"+name)
	return name == "ListPages", nil
}

func (h *stubHost) RenderModule(name string, params map[string]string, body string) (string, error) {
	h.calls = append(h.calls, "render_module:"+name)
	return `<div class="module">` + name + "|" + params["limit"] + "|" + body + `</div>`, nil
}

func (h *stubHost) RenderUser(user string, avatar bool) (string, error) {
	h.calls = append(h.calls, "render_user:"+user)
	return `<span class="user">` + user + `</span>`, nil
}

func (h *stubHost) GetPageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	h.calls = append(h.calls, "get_page_info:"+strings.Join(refs, ","))
	var out []renderer.PartialPageInfo
	for _, ref := range refs {
		if h.existing[ref] {
			title := ref
			out = append(out, renderer.PartialPageInfo{FullName: ref, Title: &title, Exists: true})
		}
	}
	return out, nil
}

func (h *stubHost) IncludePages(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.FullName+"{"+ref.Variables["a"]+"}")
	}
	h.calls = append(h.calls, "include_pages:"+strings.Join(names, ","))

	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		page := renderer.FetchedPage{FullName: ref.FullName}
		if h.existing[ref.FullName] {
			body := "**included body**"
			page.Content = &body
		}
		out = append(out, page)
	}
	return out, nil
}

func (h *stubHost) NoSuchInclude(fullName string) (string, error) {
	h.calls = append(h.calls, "no_such_include:"+fullName)
	return "[[div]]missing " + fullName + "[[/div]]", nil
}

func (h *stubHost) NormalizePageName(fullName string) (string, error) {
	return strings.ToLower(fullName), nil
}

func (h *stubHost) NextIncludeLevel() (bool, error) {
	h.calls = append(h.calls, "next_include_level")
	return true, nil
}

func (h *stubHost) GetI18nMessage(id string) (string, error) {
	h.calls = append(h.calls, "get_i18n_message:"+id)
	return "[" + id + "]", nil
}

func (h *stubHost) EvaluateExpression(expr string) (renderer.ExpressionResult, error) {
	h.calls = append(h.calls, "evaluate_expression:"+expr)
	return renderer.IntExpr(42), nil
}

func newHost() *stubHost {
	return &stubHost{existing: map[string]bool{"exists": true}}
}

func info() renderer.PageInfo {
	return renderer.PageInfo{
		Page:     "173",
		Category: "scp",
		Domain:   "example.org",
		Tags:     []string{"euclid", "keter"},
	}
}

func TestVersion(t *testing.T) {
	if got := Version(); got == "" {
		t.Error("Version() = \"\", want the ftml version")
	}
}

func TestRenderHTMLCallsBackIntoGo(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"inline formatting", "//italic// and **bold**", "<p><em>italic</em> and <strong>bold</strong></p>"},
		{"module", `[[module Rate limit="5"]]`, `<div class="module">Rate|5|</div>`},
		{"module with body", "[[module ListPages]]\nbody\n[[/module]]", `<div class="module">ListPages||body</div>`},
		{"user", "[[user someone]]", `<span class="user">someone</span>`},
		{"red link", "[[[missing|x]]]", "newpage"},
		{"blue link", "[[[exists|x]]]", `href="/exists"`},
		{"include hit", "[[include exists |a=1]]", "included body"},
		{"include miss", "[[include no-such]]", "missing no-such"},
		{"expression", "[[#expr 1 + 1]]", "42"},
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
		})
	}
}

func TestRenderHTMLReportsLinkedPages(t *testing.T) {
	got, err := New().RenderHTML(context.Background(), "[[[exists|a]]] [[[missing|b]]]", info(), newHost(), renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML() err = %v, want nil", err)
	}
	if len(got.LinkedPages) != 2 {
		t.Errorf("len(RenderHTML().LinkedPages) = %d, want 2", len(got.LinkedPages))
	}
}

func TestRenderHTMLReportsCodeBlocks(t *testing.T) {
	source := "[[code type=\"python\"]]\nprint(1)\n[[/code]]"
	got, err := New().RenderHTML(context.Background(), source, info(), newHost(), renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML() err = %v, want nil", err)
	}
	if len(got.Code) != 1 {
		t.Fatalf("len(RenderHTML().Code) = %d, want 1", len(got.Code))
	}
	if got.Code[0].Language != "python" {
		t.Errorf("RenderHTML().Code[0].Language = %q, want %q", got.Code[0].Language, "python")
	}
	if !strings.Contains(got.Code[0].Source, "print(1)") {
		t.Errorf("RenderHTML().Code[0].Source = %q, want substring %q", got.Code[0].Source, "print(1)")
	}
}

func TestRenderText(t *testing.T) {
	got, err := New().RenderText(context.Background(), "//italic// and **bold**", info(), newHost(), renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderText() err = %v, want nil", err)
	}
	if !strings.Contains(got.Body, "italic") {
		t.Errorf("RenderText() = %q, want substring %q", got.Body, "italic")
	}
	if strings.Contains(got.Body, "<em>") {
		t.Errorf("RenderText() = %q, want no HTML tags", got.Body)
	}
}

func TestCollectCodeAndHTML(t *testing.T) {
	source := "[[code type=\"go\"]]\nx := 1\n[[/code]]"
	got, err := New().CollectCodeAndHTML(context.Background(), source, info(), newHost(), renderer.ModeArticle)
	if err != nil {
		t.Fatalf("CollectCodeAndHTML() err = %v, want nil", err)
	}
	if len(got.Code) != 1 {
		t.Fatalf("len(CollectCodeAndHTML().Code) = %d, want 1", len(got.Code))
	}
	if got.Code[0].Language != "go" {
		t.Errorf("CollectCodeAndHTML().Code[0].Language = %q, want %q", got.Code[0].Language, "go")
	}
}

func TestCallbackOrderMatchesSidecar(t *testing.T) {
	host := newHost()
	if _, err := New().RenderHTML(context.Background(), "[[[exists|a]]] [[[missing|b]]]", info(), host, renderer.ModeArticle); err != nil {
		t.Fatalf("RenderHTML() err = %v, want nil", err)
	}
	want := []string{
		"next_include_level",
		"include_pages:",
		"get_page_info:exists,missing",
	}
	if len(host.calls) != len(want) {
		t.Fatalf("calls = %v, want %d entries", host.calls, len(want))
	}
	for i, call := range want {
		if host.calls[i] != call {
			t.Errorf("calls[%d] = %q, want %q", i, host.calls[i], call)
		}
	}
}

func TestRenderIsReusableAcrossCalls(t *testing.T) {
	r := New()
	for i := 0; i < 50; i++ {
		got, err := r.RenderHTML(context.Background(), "[[include exists |a=1]]\n\n[[[missing|x]]]", info(), newHost(), renderer.ModeArticle)
		if err != nil {
			t.Fatalf("RenderHTML() iteration %d err = %v, want nil", i, err)
		}
		if !strings.Contains(got.Body, "included body") {
			t.Fatalf("RenderHTML() iteration %d = %q, want substring %q", i, got.Body, "included body")
		}
	}
}
