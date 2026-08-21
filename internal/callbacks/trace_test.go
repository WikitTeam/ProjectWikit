package callbacks

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

var updateGolden = flag.Bool("update", false, "rewrite the callback trace golden file")

type tracer struct {
	inner renderer.Callbacks
	lines []string
}

func (t *tracer) log(format string, args ...any) {
	t.lines = append(t.lines, fmt.Sprintf(format, args...))
}

func (t *tracer) ModuleHasBody(name string) (bool, error) {
	t.log("module_has_body(%s)", name)
	return t.inner.ModuleHasBody(name)
}

func (t *tracer) RenderModule(name string, params map[string]string, body string) (string, error) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, params[k]))
	}
	t.log("render_module(%s, {%s}, body=%q)", name, strings.Join(pairs, ", "), body)
	return t.inner.RenderModule(name, params, body)
}

func (t *tracer) RenderUser(user string, avatar bool) (string, error) {
	t.log("render_user(%s, avatar=%t)", user, avatar)
	return t.inner.RenderUser(user, avatar)
}

func (t *tracer) GetI18nMessage(id string) (string, error) {
	t.log("get_i18n_message(%s)", id)
	return t.inner.GetI18nMessage(id)
}

func (t *tracer) GetHTMLInjectedCode(id string) (string, error) {
	t.log("get_html_injected_code(%s)", id)
	return t.inner.GetHTMLInjectedCode(id)
}

func (t *tracer) GetPageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	t.log("get_page_info([%s])", strings.Join(refs, " "))
	return t.inner.GetPageInfo(refs)
}

func (t *tracer) EvaluateExpression(expr string) (renderer.ExpressionResult, error) {
	t.log("evaluate_expression(%q)", expr)
	return t.inner.EvaluateExpression(expr)
}

func (t *tracer) NormalizePageName(fullName string) (string, error) {
	t.log("normalize_page_name(%q)", fullName)
	return t.inner.NormalizePageName(fullName)
}

func (t *tracer) IncludePages(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		keys := make([]string, 0, len(ref.Variables))
		for k := range ref.Variables {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vars := make([]string, 0, len(keys))
		for _, k := range keys {
			vars = append(vars, fmt.Sprintf("%s=%q", k, ref.Variables[k]))
		}
		parts = append(parts, fmt.Sprintf("%s{%s}", ref.FullName, strings.Join(vars, ", ")))
	}
	t.log("include_pages([%s])", strings.Join(parts, " "))
	return t.inner.IncludePages(refs)
}

func (t *tracer) NoSuchInclude(fullName string) (string, error) {
	t.log("no_such_include(%s)", fullName)
	return t.inner.NoSuchInclude(fullName)
}

func (t *tracer) NextIncludeLevel() (bool, error) {
	t.log("next_include_level()")
	return t.inner.NextIncludeLevel()
}

var traceCorpus = []struct{ name, source string }{
	{"plain", "//斜体// 和 **粗体**"},
	{"links", "[[[exists|蓝]]] 和 [[[missing|红]]]"},
	{"module-plain", "[[module Rate]]"},
	{"module-params", `[[module Rate showVotes="true" limit="5"]]`},
	{"module-needs-body-inline", `[[module ListPages perPage="20" category="scp"]]`},
	{"module-body", "[[module ListPages]]\n%%title%%\n[[/module]]"},
	{"include-hit", "[[include exists |a=1 |b=2]]"},
	{"include-miss", "[[include no-such-page]]"},
	{"user", "[[user kakushi]] 和 [[*user nobody]]"},
	{"expression", "[[#expr 1 + 1]]"},
	{"footnote", "文本[[footnote]]注[[/footnote]]"},
	{"toc", "[[toc]]\n\n+ 标题"},
	{"code", "[[code type=\"python\"]]\nprint(1)\n[[/code]]"},
}

func TestCallbackTrace(t *testing.T) {
	binary := os.Getenv(sidecar.EnvBinary)
	if binary == "" {
		t.Skipf("%s not set, skipping the callback trace test", sidecar.EnvBinary)
	}
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}

	var out strings.Builder
	for _, c := range traceCorpus {
		r, err := sidecar.New(binary)
		if err != nil {
			t.Fatalf("sidecar.New(%q) err = %v, want nil", binary, err)
		}
		tr := &tracer{inner: New(bundle.Localizer(i18n.DefaultLanguage), siteRepo{existing: map[string]bool{"exists": true}})}
		info := renderer.PageInfo{Page: "173", Category: "scp", Domain: "example.org"}
		if _, err := r.RenderHTML(context.Background(), c.source, info, tr, renderer.ModeArticle); err != nil {
			r.Close()
			t.Fatalf("RenderHTML(%s) err = %v, want nil", c.name, err)
		}
		r.Close()

		fmt.Fprintf(&out, "== %s ==\n", c.name)
		if len(tr.lines) == 0 {
			out.WriteString("(no callbacks)\n")
		}
		for _, line := range tr.lines {
			out.WriteString(line)
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}

	path := filepath.Join("testdata", "callback_trace.golden")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("MkdirAll(testdata) err = %v, want nil", err)
		}
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", path, err)
		}
		corpus := make([][2]string, 0, len(traceCorpus))
		for _, c := range traceCorpus {
			corpus = append(corpus, [2]string{c.name, c.source})
		}
		encoded, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			t.Fatalf("Marshal(corpus) err = %v, want nil", err)
		}
		corpusPath := filepath.Join("testdata", "trace_corpus.json")
		if err := os.WriteFile(corpusPath, encoded, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", corpusPath, err)
		}
		t.Logf("wrote %s and %s", path, corpusPath)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil; run go test -update to create it", path, err)
	}
	if got := out.String(); got != string(want) {
		t.Errorf("callback trace differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
