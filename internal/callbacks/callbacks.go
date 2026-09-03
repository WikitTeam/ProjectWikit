package callbacks

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/expr"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

//go:embed injected.js
var injectedCode string

const MaxIncludeLevel = 25

var (
	ErrUserNotFound = errors.New("user not found")
	ErrNoRepository = errors.New("no repository configured")
)

type ModuleError struct{ Message string }

func (e *ModuleError) Error() string { return e.Message }

type Repository interface {
	RenderModule(pc *page.Context, name string, params map[string]string, body string) (string, error)
	RenderUser(username string, avatar bool) (string, error)
	PageInfo(refs []string) ([]renderer.PartialPageInfo, error)

	// IncludeSources answers one page per ref, in the order it was asked.
	IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error)
}

type Callbacks struct {
	loc           *i18n.Localizer
	repo          Repository
	site          string
	vars          *page.Vars
	pageCtx       *page.Context
	level         int
	includeErrors map[string]bool
}

var _ renderer.Callbacks = (*Callbacks)(nil)

// SetPageVars names the page whose %%this|x%% an included page reaches. Without
// it every such name is left standing.
func (c *Callbacks) SetPageVars(vars *page.Vars) { c.vars = vars }

func (c *Callbacks) SetSite(slug string) { c.site = slug }

// SetContext hands the modules the page they are rendering into, which is what
// lets one of them redirect the whole request or set the description.
func (c *Callbacks) SetContext(pc *page.Context) { c.pageCtx = pc }

func New(loc *i18n.Localizer, repo Repository) *Callbacks {
	return &Callbacks{
		loc:           loc,
		repo:          repo,
		level:         MaxIncludeLevel,
		includeErrors: make(map[string]bool),
	}
}

func (c *Callbacks) ModuleHasBody(name string) (bool, error) {
	return module.HasContent(name), nil
}

func (c *Callbacks) ModuleIsInline(name string) (bool, error) {
	return module.IsInline(name), nil
}

func (c *Callbacks) RenderModule(name string, params map[string]string, body string) (string, error) {
	if c.repo == nil {
		return "", ErrNoRepository
	}
	lowered := make(map[string]string, len(params))
	for key, value := range params {
		lowered[strings.ToLower(key)] = value
	}
	html, err := c.repo.RenderModule(c.pageCtx, name, lowered, body)
	var moduleErr *ModuleError
	if errors.As(err, &moduleErr) {
		return `<div class="error-block"><p>` + escape.HTML(moduleErr.Message) + `</p></div>`, nil
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
		return `<span class="error-inline">` + c.text("user-not-found", "name", escape.HTML(username)) + `</span>`, nil
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
			name, _ := c.localName(ref.FullName)
			c.includeErrors[wikidot.Normalize(name)] = true
			out = append(out, renderer.FetchedPage{FullName: ref.FullName})
		}
		return out, nil
	}
	if c.repo == nil {
		return nil, ErrNoRepository
	}

	local := make([]renderer.IncludeRef, 0, len(refs))
	asked := make([]string, 0, len(refs))
	for _, ref := range refs {
		name, ok := c.localName(ref.FullName)
		if !ok {
			continue
		}
		asked = append(asked, ref.FullName)
		ref.FullName = name
		local = append(local, ref)
	}
	fetched, err := c.repo.IncludeSources(local)
	if err != nil {
		return nil, err
	}
	// ftml pairs the answer with the request by name, so a ref that named this
	// wiki has to go back carrying the prefix it arrived with.
	for i := range fetched {
		if i < len(asked) {
			fetched[i].FullName = asked[i]
		}
	}
	// %%this|x%% in an included page names the including page, so the pass runs
	// here, on the way in, rather than wherever the include came from.
	for i := range fetched {
		if fetched[i].Content == nil {
			continue
		}
		substituted := page.PreRender(*fetched[i].Content, c.vars)
		fetched[i].Content = &substituted
	}
	return fetched, nil
}

// A ref naming this very wiki means the same as one naming no wiki. One naming
// another is turned away, since looking it up here reads the site as a category.
func (c *Callbacks) localName(fullName string) (string, bool) {
	rest, prefixed := strings.CutPrefix(fullName, ":")
	if !prefixed {
		return fullName, true
	}
	slug, name, found := strings.Cut(rest, ":")
	if !found || c.site == "" || !strings.EqualFold(slug, c.site) {
		return "", false
	}
	return name, true
}

func (c *Callbacks) NoSuchInclude(fullName string) (string, error) {
	name, ok := c.localName(fullName)
	if !ok {
		return `[[div class="error-block"]]` + c.text("include-off-site", "name", fullName) + `[[/div]]`, nil
	}
	if c.includeErrors[wikidot.Normalize(name)] {
		return `[[div class="error-block"]]` + c.text("include-loop", "name", name) + `[[/div]]`, nil
	}
	return `[[div class="error-block"]]` + c.text("include-missing", "name", name) +
		` ([[a href="/` + name + `/edit/true" target="_blank"]]` + c.text("include-create") + `[[/a]])[[/div]]`, nil
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
