package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/article"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const templateName = "_template"

type Preview struct {
	deps Deps
}

var _ http.Handler = (*Preview)(nil)

func NewPreview(d Deps) *Preview { return &Preview{deps: d} }

type previewCall struct {
	PageID     string          `json:"pageId"`
	Title      string          `json:"title"`
	Source     string          `json:"source"`
	PathParams json.RawMessage `json:"pathParams"`
}

func (h *Preview) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, field("error", "method not allowed"))
		return
	}
	loc := h.deps.Bundle.For(r.Context())

	current := site.FromContext(r.Context())
	if current == nil {
		h.deps.log().Error("preview", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}

	var parsed previewCall
	raw, _, err := peek(r.Body)
	if err != nil || json.Unmarshal(raw, &parsed) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if parsed.PageID == "" || !wikidot.NameAllowed(parsed.PageID) {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-page-id")))
		return
	}

	body, styles, err := h.render(r, loc, current, parsed)
	if err != nil {
		h.deps.log().Error("preview", "page", parsed.PageID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	out, err := wikijson.Marshal(wikijson.Object{
		{Key: "title", Value: parsed.Title},
		{Key: "content", Value: body},
		{Key: "style", Value: strings.Join(styles, "")},
		{Key: "styles", Value: styles},
	})
	if err != nil {
		h.deps.log().Error("preview", "page", parsed.PageID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Preview) render(r *http.Request, loc *i18n.Localizer, current *db.Site, parsed previewCall) (string, []string, error) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	params := page.ParsePathParams(string(parsed.PathParams))

	found, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), parsed.PageID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", nil, err
	}

	source, err := h.template(ctx, found, parsed.PageID)
	if err != nil {
		return "", nil, err
	}

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(ctx, loc, current, user)
	vars := env.Vars(found)
	fill := vars
	if fill == nil {
		fill = page.NewDraftVars(strings.ToLower(parsed.PageID), parsed.Title, user, repo.NewVarSource(ctx, h.deps.DB, current), loc)
	}
	fill.SetContent(parsed.Source)

	source = page.OutsideModules(source, module.HasItemBody, func(s string) string {
		return page.PageVars(s, fill, 1, 1)
	})
	source = page.ApplyTemplate(source, article.ThisPage(params, canonical(current, found, parsed, params)))
	source = form.Strip(source)
	source = page.PreRender(source, vars)

	info, err := env.PageInfo(found)
	if err != nil {
		return "", nil, err
	}
	pc := page.NewContext(found, found, params, user)
	pc.CSRF, _ = csrf.Token(r)
	pc.Draft = found == nil

	html, err := env.HTML(source, info, env.Callbacks(vars, pc), renderer.ModeArticle)
	if err != nil {
		return "", nil, err
	}
	return html.Body, pc.ComputedStyles, nil
}

// A page with no row of its own still gets the category template, which is what
// the editor shows while the page is being created.
func (h *Preview) template(ctx context.Context, found *db.Article, pageID string) (string, error) {
	category, name := wikidot.Split(strings.ToLower(pageID))
	if found != nil {
		category, name = found.Category, found.Name
	}
	if name == templateName {
		return "%%content%%", nil
	}
	template, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), category+":"+templateName)
	if errors.Is(err, db.ErrNotFound) {
		return "%%content%%", nil
	}
	if err != nil {
		return "", err
	}
	source, err := h.deps.DB.LatestSource(ctx, template.ID)
	if errors.Is(err, db.ErrNotFound) {
		return "%%content%%", nil
	}
	if err != nil {
		return "", err
	}
	return page.TemplateLayout(source), nil
}

func canonical(current *db.Site, found *db.Article, parsed previewCall, params page.PathParams) string {
	name := parsed.PageID
	if found != nil {
		name = found.FullName()
	}
	return "//" + current.Domain + "/" + name + article.Encode(params)
}
