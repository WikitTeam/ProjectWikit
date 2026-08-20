package callbacks

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

type siteRepo struct {
	existing map[string]bool
}

func (r siteRepo) RenderModule(name string, params map[string]string, body string) (string, error) {
	return `<div class="module">` + name + "|" + body + `</div>`, nil
}

func (r siteRepo) RenderUser(username string, avatar bool) (string, error) {
	if username != "kakushi" {
		return "", ErrUserNotFound
	}
	return `<span class="user">kakushi</span>`, nil
}

func (r siteRepo) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	var out []renderer.PartialPageInfo
	for _, ref := range refs {
		if r.existing[ref] {
			title := ref
			out = append(out, renderer.PartialPageInfo{FullName: ref, Exists: true, Title: &title})
		}
	}
	return out, nil
}

func (r siteRepo) IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		page := renderer.FetchedPage{FullName: ref.FullName}
		if r.existing[ref.FullName] {
			body := "**被包含的内容**"
			page.Content = &body
		}
		out = append(out, page)
	}
	return out, nil
}

func renderWith(t *testing.T, source string) string {
	t.Helper()
	binary := os.Getenv(sidecar.EnvBinary)
	if binary == "" {
		t.Skipf("未设置 %s，跳过真实渲染链路测试", sidecar.EnvBinary)
	}
	r, err := sidecar.New(binary)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v，期望 nil", binary, err)
	}
	t.Cleanup(func() { r.Close() })

	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v，期望 nil", err)
	}
	cb := New(bundle.Localizer(i18n.DefaultLanguage), siteRepo{existing: map[string]bool{"exists": true}})

	info := renderer.PageInfo{Page: "173", Category: "scp", Domain: "example.org"}
	got, err := r.RenderHTML(context.Background(), source, info, cb, renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML(%q) err = %v，期望 nil", source, err)
	}
	return got.Body
}

func TestRealRenderUsesCallbacks(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"模块", "[[module Rate]]", `<div class="module">Rate|</div>`},
		{"带 body 的模块", "[[module CSS]]div{}[[/module]]", `<div class="module">CSS|div{}</div>`},
		{"红链", "[[[missing|红]]]", `class="newpage"`},
		{"已存在的链接不是红链", "[[[exists|蓝]]]", `href="/exists"`},
		{"用户", "[[user kakushi]]", `<span class="user">kakushi</span>`},
		{"用户不存在", "[[user nobody]]", "用户 'nobody' 不存在"},
		{"include 缺失", "[[include :other:page]]", "不存在"},
		{"include 存在", "[[include exists]]", "<strong>被包含的内容</strong>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderWith(t, tt.source)
			if !strings.Contains(got, tt.want) {
				t.Errorf("RenderHTML(%q) = %q，期望包含 %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestRealRenderUsesI18nCatalog(t *testing.T) {
	got := renderWith(t, "正文[[footnote]]脚注内容[[/footnote]]")
	if !strings.Contains(got, "脚注") {
		t.Errorf("RenderHTML() = %q，期望包含 %q", got, "脚注")
	}
}
