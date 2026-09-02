// Package pagerender assembles the engine, the modules and the data layer into
// the callbacks one request renders with.
package pagerender

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type Deps struct {
	DB     *db.DB
	Engine renderer.Renderer
	Icons  roles.IconLoader
	Trust  *proxyheader.Trust
}

type Env struct {
	deps Deps
	ctx  context.Context
	loc  *i18n.Localizer
	site *db.Site
	user *db.User
	ip   *netip.Addr
}

func (d Deps) Env(ctx context.Context, loc *i18n.Localizer, site *db.Site, user *db.User) *Env {
	return &Env{deps: d, ctx: ctx, loc: loc, site: site, user: user}
}

// Only the layer holding the proxy trust can say which forwarded address to
// believe.
func (e *Env) SetClient(r *http.Request) {
	if e.deps.Trust == nil {
		return
	}
	if addr, ok := e.deps.Trust.ClientIP(r); ok {
		e.ip = &addr
	}
}

func (e *Env) HTML(source string, info renderer.PageInfo, cb *callbacks.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return e.deps.Engine.RenderHTML(e.ctx, source, info, cb, mode)
}

func (e *Env) Text(source string, info renderer.PageInfo, cb *callbacks.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return e.deps.Engine.RenderText(e.ctx, source, info, cb, mode)
}

func (e *Env) CodeAndHTML(source string, info renderer.PageInfo, cb *callbacks.Callbacks, mode renderer.Mode) (renderer.Parts, error) {
	return e.deps.Engine.CollectCodeAndHTML(e.ctx, source, info, cb, mode)
}

func (e *Env) Vars(of *db.Article) *page.Vars {
	if of == nil {
		return nil
	}
	return page.NewVars(of, e.user, repo.NewVarSource(e.ctx, e.deps.DB, e.site), e.loc)
}

func (e *Env) PageInfo(source *db.Article) (renderer.PageInfo, error) {
	info := renderer.PageInfo{
		Site:        e.site.Slug,
		Domain:      e.site.Domain,
		MediaDomain: e.site.MediaDomain,
	}
	if source == nil {
		return info, nil
	}
	info.Page = source.Name
	info.Category = source.Category
	tags, err := e.deps.DB.ArticleTagNames(e.ctx, source.ID)
	if err != nil {
		return renderer.PageInfo{}, err
	}
	info.Tags = tags
	return info, nil
}

func (e *Env) Callbacks(vars *page.Vars, pc *page.Context) *callbacks.Callbacks {
	cb := callbacks.New(e.loc, e.store())
	cb.SetPageVars(vars)
	cb.SetContext(pc)
	return cb
}

func (e *Env) ModuleAPI(pc *page.Context, name, method string, params map[string]string) (wikijson.Object, error) {
	return e.store().CallAPI(pc, name, method, params)
}

func (e *Env) store() *repo.Repository {
	users := printuser.New(e.loc, e.deps.Icons)
	return repo.New(e.ctx, e.deps.DB, users, repo.Options{
		Loc:               e.loc,
		Site:              e.site,
		User:              e.user,
		Render:            e.nested,
		RenderMessage:     e.message,
		RenderMessageText: e.messageText,
		Vars:              repo.NewVarSource(e.ctx, e.deps.DB, e.site),
		ClientIP:          e.ip,
	})
}

// The page it renders against is the one the module put in the context, which
// for a listing is the row and not the page the listing sits on.
func (e *Env) nested(source string, pc *page.Context) (string, error) {
	vars := e.Vars(pc.Article)
	info, err := e.PageInfo(pc.SourceArticle)
	if err != nil {
		return "", err
	}
	html, err := e.HTML(page.PreRender(source, vars), info, e.Callbacks(vars, pc), renderer.ModeArticle)
	if err != nil {
		return "", err
	}
	return html.Body, nil
}

func (e *Env) message(source string) (string, error) {
	pc := page.NewContext(nil, nil, nil, e.user)
	info, err := e.PageInfo(nil)
	if err != nil {
		return "", err
	}
	html, err := e.HTML(page.PreRender(source, nil), info, e.Callbacks(nil, pc), renderer.ModeMessage)
	if err != nil {
		return "", err
	}
	return html.Body, nil
}

func (e *Env) messageText(source string) (string, error) {
	pc := page.NewContext(nil, nil, nil, e.user)
	info, err := e.PageInfo(nil)
	if err != nil {
		return "", err
	}
	text, err := e.Text(page.PreRender(source, nil), info, e.Callbacks(nil, pc), renderer.ModeMessage)
	if err != nil {
		return "", err
	}
	return text.Body, nil
}
