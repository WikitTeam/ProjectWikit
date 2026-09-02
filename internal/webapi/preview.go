package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/WikitTeam/ProjectWikit/internal/article"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
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
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)

	current := site.FromContext(r.Context())
	if current == nil {
		h.deps.log().Error("preview", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-csrf-failed")))
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

	body, style, err := h.render(r, loc, current, parsed)
	if err != nil {
		h.deps.log().Error("preview", "page", parsed.PageID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	out, err := wikijson.Marshal(wikijson.Object{
		{Key: "title", Value: parsed.Title},
		{Key: "content", Value: body},
		{Key: "style", Value: style},
	})
	if err != nil {
		h.deps.log().Error("preview", "page", parsed.PageID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Preview) render(r *http.Request, loc *i18n.Localizer, current *db.Site, parsed previewCall) (string, string, error) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	params := page.ParsePathParams(string(parsed.PathParams))

	found, err := h.deps.DB.ArticleByName(ctx, parsed.PageID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", "", err
	}

	source, err := h.template(ctx, found)
	if err != nil {
		return "", "", err
	}

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(ctx, loc, current, user)
	vars := env.Vars(found)
	vars.SetContent(parsed.Source)

	source = page.PageVars(source, vars, 1, 1)
	source = page.ApplyTemplate(source, article.ThisPage(params, canonical(current, found, parsed, params)))
	source = form.Strip(source)
	source = page.PreRender(source, vars)

	info, err := env.PageInfo(found)
	if err != nil {
		return "", "", err
	}
	pc := page.NewContext(found, found, params, user)
	pc.CSRF, _ = csrf.Token(r)

	html, err := env.HTML(source, info, env.Callbacks(vars, pc), renderer.ModeArticle)
	if err != nil {
		return "", "", err
	}
	return html.Body, pc.ComputedStyle, nil
}

// A page with no row of its own still gets the category template, which is what
// the editor shows while the page is being created.
func (h *Preview) template(ctx context.Context, found *db.Article) (string, error) {
	if found == nil || found.Name == templateName {
		return "%%content%%", nil
	}
	template, err := h.deps.DB.ArticleByName(ctx, found.Category+":"+templateName)
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
	return source, nil
}

func canonical(current *db.Site, found *db.Article, parsed previewCall, params page.PathParams) string {
	name := parsed.PageID
	if found != nil {
		name = found.FullName()
	}
	return "//" + current.Domain + "/" + name + article.Encode(params)
}
