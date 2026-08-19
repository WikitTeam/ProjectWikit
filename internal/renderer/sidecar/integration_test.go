package sidecar

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type stubHost struct {
	renderer.NopCallbacks
	existing map[string]bool
}

func (h stubHost) RenderModule(name string, _ map[string]string, _ string) (string, error) {
	return `<div class="module">` + name + `</div>`, nil
}

func (h stubHost) NormalizePageName(name string) (string, error) {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-")), nil
}

func (h stubHost) GetPageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	var out []renderer.PartialPageInfo
	for _, r := range refs {
		if h.existing[r] {
			title := r
			out = append(out, renderer.PartialPageInfo{FullName: r, Exists: true, Title: &title})
		}
	}
	return out, nil
}

func (h stubHost) NoSuchInclude(fullName string) (string, error) {
	return "[[include " + fullName + " 不存在]]", nil
}

func newReal(t *testing.T) *Renderer {
	t.Helper()
	bin := os.Getenv(EnvBinary)
	if bin == "" {
		t.Skipf("未设置 %s，跳过真实 sidecar 集成测试", EnvBinary)
	}
	r, err := New(bin)
	if err != nil {
		t.Fatalf("New(%q) err = %v，期望 nil", bin, err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestRealSidecarRendersHTML(t *testing.T) {
	r := newReal(t)
	host := stubHost{existing: map[string]bool{"exists": true}}
	info := renderer.PageInfo{Page: "173", Category: "scp", Domain: "example.org"}

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"行内格式", "//斜体// 和 **粗体**", "<p><em>斜体</em> 和 <strong>粗体</strong></p>"},
		{"模块", "[[module Rate]]", `<div class="module">Rate</div>`},
		{"注释被去掉", "[!-- 注释 --]文本", "<p>文本</p>"},
		{"include 缺失", "[[include :other:page]]", "不存在"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.RenderHTML(context.Background(), tt.source, info, host, renderer.ModeArticle)
			if err != nil {
				t.Fatalf("RenderHTML(%q) err = %v，期望 nil", tt.source, err)
			}
			if !strings.Contains(got.Body, tt.want) {
				t.Errorf("RenderHTML(%q) = %q，期望包含 %q", tt.source, got.Body, tt.want)
			}
		})
	}
}

func TestRealSidecarMarksRedLinks(t *testing.T) {
	r := newReal(t)
	host := stubHost{existing: map[string]bool{"exists": true}}
	info := renderer.PageInfo{Page: "173", Domain: "example.org"}

	got, err := r.RenderHTML(context.Background(), "[[[exists|蓝]]] [[[missing|红]]]", info, host, renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML() err = %v，期望 nil", err)
	}
	if !strings.Contains(got.Body, `<a href="/missing" class="newpage">红</a>`) {
		t.Errorf("Body = %q，期望缺失页链接带 class=newpage", got.Body)
	}
	if strings.Contains(got.Body, `<a href="/exists" class="newpage">`) {
		t.Errorf("Body = %q，期望已存在的页面不带 class=newpage", got.Body)
	}
	if len(got.LinkedPages) != 2 {
		t.Errorf("LinkedPages = %v，期望两项", got.LinkedPages)
	}
}

func TestRealSidecarRendersText(t *testing.T) {
	r := newReal(t)
	host := stubHost{}

	got, err := r.RenderText(context.Background(), "**粗体** 文本", renderer.PageInfo{Domain: "example.org"}, host, renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderText() err = %v，期望 nil", err)
	}
	if strings.TrimSpace(got.Body) != "粗体 文本" {
		t.Errorf("RenderText() = %q，期望 %q", got.Body, "粗体 文本")
	}
}

func TestRealSidecarReusesProcess(t *testing.T) {
	r := newReal(t)
	host := stubHost{}
	info := renderer.PageInfo{Domain: "example.org"}

	for i := range 3 {
		got, err := r.RenderHTML(context.Background(), "**x**", info, host, renderer.ModeArticle)
		if err != nil {
			t.Fatalf("第 %d 次 RenderHTML() err = %v，期望 nil", i+1, err)
		}
		if !strings.Contains(got.Body, "<strong>x</strong>") {
			t.Errorf("第 %d 次 = %q，期望包含 <strong>x</strong>", i+1, got.Body)
		}
	}
}
