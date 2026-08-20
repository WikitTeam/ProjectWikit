package callbacks

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/expr"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/modules"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

//go:embed injected.js
var injectedCode string

const MaxIncludeLevel = 25

var (
	ErrUserNotFound = errors.New("用户不存在")
	ErrNoRepository = errors.New("没有配置数据源")
)

type ModuleError struct{ Message string }

func (e *ModuleError) Error() string { return e.Message }

type Repository interface {
	RenderModule(name string, params map[string]string, body string) (string, error)
	RenderUser(username string, avatar bool) (string, error)
	PageInfo(refs []string) ([]renderer.PartialPageInfo, error)
	IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error)
}

type Callbacks struct {
	loc           *i18n.Localizer
	repo          Repository
	level         int
	includeErrors map[string]bool
}

var _ renderer.Callbacks = (*Callbacks)(nil)

func New(loc *i18n.Localizer, repo Repository) *Callbacks {
	return &Callbacks{
		loc:           loc,
		repo:          repo,
		level:         MaxIncludeLevel,
		includeErrors: make(map[string]bool),
	}
}

func (c *Callbacks) ModuleHasBody(name string) (bool, error) {
	return modules.HasContent(name), nil
}

func (c *Callbacks) RenderModule(name string, params map[string]string, body string) (string, error) {
	if c.repo == nil {
		return "", ErrNoRepository
	}
	lowered := make(map[string]string, len(params))
	for key, value := range params {
		lowered[strings.ToLower(key)] = value
	}
	html, err := c.repo.RenderModule(name, lowered, body)
	var moduleErr *ModuleError
	if errors.As(err, &moduleErr) {
		return `<div class="error-block"><p>` + djangoEscape(moduleErr.Message) + `</p></div>`, nil
	}
	if err != nil {
		return "", err
	}
	return html, nil
}

func (c *Callbacks) RenderUser(username string, avatar bool) (string, error) {
	if c.repo == nil {
		return "", ErrNoRepository
	}
	html, err := c.repo.RenderUser(username, avatar)
	if errors.Is(err, ErrUserNotFound) {
		return `<span class="error-inline">` + c.text("user-not-found", "name", djangoEscape(username)) + `</span>`, nil
	}
	if err != nil {
		return "", err
	}
	return html, nil
}

func (c *Callbacks) GetI18nMessage(id string) (string, error) {
	return c.text(id), nil
}

func (c *Callbacks) GetHTMLInjectedCode(id string) (string, error) {
	encoded, err := json.Marshal(id)
	if err != nil {
		return "", err
	}
	return strings.Replace(injectedCode, "%s", string(encoded), 1), nil
}

func (c *Callbacks) GetPageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	if c.repo == nil {
		return nil, ErrNoRepository
	}
	return c.repo.PageInfo(refs)
}

func (c *Callbacks) EvaluateExpression(source string) (renderer.ExpressionResult, error) {
	v := expr.Evaluate(source)
	switch v.Kind {
	case expr.KindFloat:
		return renderer.FloatExpr(v.Float), nil
	case expr.KindInt, expr.KindBool:
		return renderer.IntExpr(v.AsInt()), nil
	case expr.KindStr:
		return renderer.StringExpr(v.Str), nil
	}
	return renderer.ExpressionResult{}, nil
}

func (c *Callbacks) NormalizePageName(fullName string) (string, error) {
	return wikidot.Normalize(fullName), nil
}

func (c *Callbacks) IncludePages(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	if c.level <= 0 {
		out := make([]renderer.FetchedPage, 0, len(refs))
		for _, ref := range refs {
			c.includeErrors[wikidot.Normalize(ref.FullName)] = true
			out = append(out, renderer.FetchedPage{FullName: ref.FullName})
		}
		return out, nil
	}
	if c.repo == nil {
		return nil, ErrNoRepository
	}
	return c.repo.IncludeSources(refs)
}

func (c *Callbacks) NoSuchInclude(fullName string) (string, error) {
	if c.includeErrors[wikidot.Normalize(fullName)] {
		return `[[div class="error-block"]]` + c.text("include-loop", "name", fullName) + `[[/div]]`, nil
	}
	return `[[div class="error-block"]]` + c.text("include-missing", "name", fullName) +
		` ([[a href="/` + fullName + `/edit/true" target="_blank"]]` + c.text("include-create") + `[[/a]])[[/div]]`, nil
}

func (c *Callbacks) NextIncludeLevel() (bool, error) {
	if c.level <= 0 {
		return false, nil
	}
	c.level--
	return true, nil
}

func (c *Callbacks) text(id string, args ...any) string {
	if c.loc == nil {
		return id
	}
	return c.loc.T(id, args...)
}

var escaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
)

func djangoEscape(s string) string { return escaper.Replace(s) }
