package callbacks

import (
	"errors"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type fakeRepo struct {
	moduleErr error
	userErr   error

	moduleName  string
	moduleParam map[string]string
	includeSeen []renderer.IncludeRef
	includeBody string
}

func (r *fakeRepo) RenderModule(name string, params map[string]string, body string) (string, error) {
	r.moduleName = name
	r.moduleParam = params
	if r.moduleErr != nil {
		return "", r.moduleErr
	}
	return "<div>" + name + "</div>", nil
}

func (r *fakeRepo) RenderUser(username string, avatar bool) (string, error) {
	if r.userErr != nil {
		return "", r.userErr
	}
	return "<span>" + username + "</span>", nil
}

func (r *fakeRepo) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	out := make([]renderer.PartialPageInfo, 0, len(refs))
	for _, ref := range refs {
		out = append(out, renderer.PartialPageInfo{FullName: ref, Exists: true})
	}
	return out, nil
}

func (r *fakeRepo) IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	r.includeSeen = refs
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		body := r.includeBody
		if body == "" {
			body = "内容"
		}
		out = append(out, renderer.FetchedPage{FullName: ref.FullName, Content: &body})
	}
	return out, nil
}

func newCallbacks(t *testing.T, repo Repository) *Callbacks {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return New(bundle.Localizer(i18n.DefaultLanguage), repo)
}

func TestModuleHasBody(t *testing.T) {
	c := newCallbacks(t, nil)
	tests := []struct {
		name string
		want bool
	}{
		{"css", true},
		{"CSS", true},
		{"rate", false},
		{"interwiki", false},
		{"nosuchmodule", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.ModuleHasBody(tt.name)
			if err != nil {
				t.Fatalf("ModuleHasBody(%q) err = %v, want nil", tt.name, err)
			}
			if got != tt.want {
				t.Errorf("ModuleHasBody(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRenderModuleLowercasesParamKeys(t *testing.T) {
	repo := &fakeRepo{}
	c := newCallbacks(t, repo)

	if _, err := c.RenderModule("Rate", map[string]string{"ShowVotes": "yes"}, ""); err != nil {
		t.Fatalf("RenderModule() err = %v, want nil", err)
	}
	if repo.moduleName != "Rate" {
		t.Errorf("moduleName = %q, want %q", repo.moduleName, "Rate")
	}
	if _, ok := repo.moduleParam["showvotes"]; !ok {
		t.Errorf("params = %v, want key %q", repo.moduleParam, "showvotes")
	}
}

func TestRenderModuleTurnsModuleErrorIntoErrorBlock(t *testing.T) {
	repo := &fakeRepo{moduleErr: &ModuleError{Message: `坏了 <b>&`}}
	c := newCallbacks(t, repo)

	got, err := c.RenderModule("rate", nil, "")
	if err != nil {
		t.Fatalf("RenderModule() err = %v, want nil", err)
	}
	want := `<div class="error-block"><p>坏了 &lt;b&gt;&amp;</p></div>`
	if got != want {
		t.Errorf("RenderModule() = %q, want %q", got, want)
	}
}

func TestRenderModulePropagatesOtherErrors(t *testing.T) {
	boom := errors.New("boom")
	c := newCallbacks(t, &fakeRepo{moduleErr: boom})

	if _, err := c.RenderModule("rate", nil, ""); !errors.Is(err, boom) {
		t.Errorf("RenderModule() err = %v, want %v", err, boom)
	}
}

func TestRenderUserNotFound(t *testing.T) {
	c := newCallbacks(t, &fakeRepo{userErr: ErrUserNotFound})

	got, err := c.RenderUser("kaku<shi", false)
	if err != nil {
		t.Fatalf("RenderUser() err = %v, want nil", err)
	}
	want := `<span class="error-inline">用户 'kaku&lt;shi' 不存在</span>`
	if got != want {
		t.Errorf("RenderUser() = %q, want %q", got, want)
	}
}

func TestGetI18nMessage(t *testing.T) {
	c := newCallbacks(t, nil)
	tests := []struct {
		id   string
		want string
	}{
		{"button-copy-clipboard", "复制"},
		{"toc-open", "展开"},
		{"no-such-message", "no-such-message"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, err := c.GetI18nMessage(tt.id)
			if err != nil {
				t.Fatalf("GetI18nMessage(%q) err = %v, want nil", tt.id, err)
			}
			if got != tt.want {
				t.Errorf("GetI18nMessage(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestGetHTMLInjectedCodeSubstitutesIDOnce(t *testing.T) {
	c := newCallbacks(t, nil)

	got, err := c.GetHTMLInjectedCode("abc-12")
	if err != nil {
		t.Fatalf("GetHTMLInjectedCode() err = %v, want nil", err)
	}
	if !strings.Contains(got, `id: "abc-12"`) {
		t.Errorf("output has no %q", `id: "abc-12"`)
	}
	if strings.Contains(got, "%s") {
		t.Errorf("output still contains placeholder %q, want it substituted", "%s")
	}
	if !strings.HasPrefix(got, "\n    <script>") {
		t.Errorf("output prefix = %q, want %q", got[:min(len(got), 16)], "\n    <script>")
	}
}

func TestEvaluateExpressionReportsBoolAsInt(t *testing.T) {
	c := newCallbacks(t, nil)
	tests := []struct {
		src  string
		want renderer.ExpressionResult
	}{
		{"1 + 1", renderer.IntExpr(2)},
		{"1.5", renderer.FloatExpr(1.5)},
		{"'x'", renderer.StringExpr("x")},
		{"1 == 1", renderer.IntExpr(1)},
		{"1 == 2", renderer.IntExpr(0)},
		{"nonsense", renderer.ExpressionResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got, err := c.EvaluateExpression(tt.src)
			if err != nil {
				t.Fatalf("EvaluateExpression(%q) err = %v, want nil", tt.src, err)
			}
			if got != tt.want {
				t.Errorf("EvaluateExpression(%q) = %+v, want %+v", tt.src, got, tt.want)
			}
		})
	}
}

func TestNormalizePageName(t *testing.T) {
	c := newCallbacks(t, nil)
	got, err := c.NormalizePageName("SCP 173")
	if err != nil {
		t.Fatalf("NormalizePageName() err = %v, want nil", err)
	}
	if got != "scp-173" {
		t.Errorf("NormalizePageName(%q) = %q, want %q", "SCP 173", got, "scp-173")
	}
}

func TestNextIncludeLevelStopsAtZero(t *testing.T) {
	c := newCallbacks(t, &fakeRepo{})
	for i := range MaxIncludeLevel {
		ok, err := c.NextIncludeLevel()
		if err != nil {
			t.Fatalf("call %d: NextIncludeLevel() err = %v, want nil", i+1, err)
		}
		if !ok {
			t.Fatalf("call %d: NextIncludeLevel() = false, want true", i+1)
		}
	}
	ok, err := c.NextIncludeLevel()
	if err != nil {
		t.Fatalf("NextIncludeLevel() err = %v, want nil", err)
	}
	if ok {
		t.Errorf("call %d: NextIncludeLevel() = true, want false", MaxIncludeLevel+1)
	}
}

func TestIncludePagesReturnsEmptyContentAfterOverflow(t *testing.T) {
	repo := &fakeRepo{}
	c := newCallbacks(t, repo)
	for range MaxIncludeLevel {
		c.NextIncludeLevel()
	}

	got, err := c.IncludePages([]renderer.IncludeRef{{FullName: "SCP 173"}})
	if err != nil {
		t.Fatalf("IncludePages() err = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(IncludePages()) = %d, want 1", len(got))
	}
	if got[0].Content != nil {
		t.Errorf("Content = %v, want nil", *got[0].Content)
	}
	if repo.includeSeen != nil {
		t.Error("repository was called after include overflow, want no call")
	}
}

func TestNoSuchIncludeReportsMissing(t *testing.T) {
	c := newCallbacks(t, &fakeRepo{})

	got, err := c.NoSuchInclude("scp-173")
	if err != nil {
		t.Fatalf("NoSuchInclude() err = %v, want nil", err)
	}
	want := `[[div class="error-block"]]插入的页面 "scp-173" 不存在 ([[a href="/scp-173/edit/true" target="_blank"]]立刻创建[[/a]])[[/div]]`
	if got != want {
		t.Errorf("NoSuchInclude() = %q, want %q", got, want)
	}
}

func TestNoSuchIncludeReportsLoopAfterOverflow(t *testing.T) {
	c := newCallbacks(t, &fakeRepo{})
	for range MaxIncludeLevel {
		c.NextIncludeLevel()
	}
	if _, err := c.IncludePages([]renderer.IncludeRef{{FullName: "SCP 173"}}); err != nil {
		t.Fatalf("IncludePages() err = %v, want nil", err)
	}

	got, err := c.NoSuchInclude("SCP 173")
	if err != nil {
		t.Fatalf("NoSuchInclude() err = %v, want nil", err)
	}
	want := `[[div class="error-block"]]插入的页面 "SCP 173" 导致了无限包含循环[[/div]]`
	if got != want {
		t.Errorf("NoSuchInclude() = %q, want %q", got, want)
	}
}

func TestRepositoryBoundCallbacksFailWithoutRepository(t *testing.T) {
	c := newCallbacks(t, nil)

	if _, err := c.RenderModule("rate", nil, ""); !errors.Is(err, ErrNoRepository) {
		t.Errorf("RenderModule() err = %v, want ErrNoRepository", err)
	}
	if _, err := c.RenderUser("kakushi", false); !errors.Is(err, ErrNoRepository) {
		t.Errorf("RenderUser() err = %v, want ErrNoRepository", err)
	}
	if _, err := c.GetPageInfo([]string{"a"}); !errors.Is(err, ErrNoRepository) {
		t.Errorf("GetPageInfo() err = %v, want ErrNoRepository", err)
	}
	if _, err := c.IncludePages([]renderer.IncludeRef{{FullName: "a"}}); !errors.Is(err, ErrNoRepository) {
		t.Errorf("IncludePages() err = %v, want ErrNoRepository", err)
	}
}

func TestPureCallbacksWorkWithoutRepository(t *testing.T) {
	c := newCallbacks(t, nil)

	if _, err := c.ModuleHasBody("css"); err != nil {
		t.Errorf("ModuleHasBody() err = %v, want nil", err)
	}
	if _, err := c.GetI18nMessage("toc-open"); err != nil {
		t.Errorf("GetI18nMessage() err = %v, want nil", err)
	}
	if _, err := c.GetHTMLInjectedCode("x"); err != nil {
		t.Errorf("GetHTMLInjectedCode() err = %v, want nil", err)
	}
	if _, err := c.EvaluateExpression("1"); err != nil {
		t.Errorf("EvaluateExpression() err = %v, want nil", err)
	}
	if _, err := c.NormalizePageName("x"); err != nil {
		t.Errorf("NormalizePageName() err = %v, want nil", err)
	}
	if _, err := c.NoSuchInclude("x"); err != nil {
		t.Errorf("NoSuchInclude() err = %v, want nil", err)
	}
	if _, err := c.NextIncludeLevel(); err != nil {
		t.Errorf("NextIncludeLevel() err = %v, want nil", err)
	}
}

func TestIncludePagesSubstitutesThisVars(t *testing.T) {
	repo := &fakeRepo{includeBody: "before %%this|title%% after"}
	c := newCallbacks(t, repo)
	c.SetPageVars(page.NewVars(&db.Article{Title: "Host Title"}, nil, nil, nil))

	got, err := c.IncludePages([]renderer.IncludeRef{{FullName: "component:box"}})
	if err != nil {
		t.Fatalf("IncludePages() err = %v, want nil", err)
	}
	if want := "before Host Title after"; *got[0].Content != want {
		t.Errorf("Content = %q, want %q", *got[0].Content, want)
	}
}

func TestIncludePagesWithoutPageVarsKeepsThisVars(t *testing.T) {
	repo := &fakeRepo{includeBody: "before %%this|title%% after"}
	c := newCallbacks(t, repo)

	got, err := c.IncludePages([]renderer.IncludeRef{{FullName: "component:box"}})
	if err != nil {
		t.Fatalf("IncludePages() err = %v, want nil", err)
	}
	if want := "before %%this|title%% after"; *got[0].Content != want {
		t.Errorf("Content = %q, want %q", *got[0].Content, want)
	}
}
