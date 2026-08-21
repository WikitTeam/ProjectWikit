package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type tracer struct {
	inner renderer.Callbacks
	lines []string
}

var _ renderer.Callbacks = (*tracer)(nil)

func (t *tracer) log(format string, args ...any) {
	t.lines = append(t.lines, fmt.Sprintf(format, args...))
}

func (t *tracer) ModuleHasBody(name string) (bool, error) {
	t.log("module_has_body(%s)", name)
	return t.inner.ModuleHasBody(name)
}

func (t *tracer) RenderModule(name string, params map[string]string, body string) (string, error) {
	t.log("render_module(%s, {%s}, body=%q)", name, joinPairs(params), body)
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
		parts = append(parts, fmt.Sprintf("%s{%s}", ref.FullName, joinPairs(ref.Variables)))
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

func joinPairs(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%q", key, values[key]))
	}
	return strings.Join(pairs, ", ")
}
