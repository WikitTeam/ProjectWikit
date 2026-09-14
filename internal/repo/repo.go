// Package repo composes the data layer into the renderer's Repository.
package repo

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/module"

	// The modules register themselves; without this nothing answers a
	// [[module]] and every page carrying one draws an error block instead.
	_ "github.com/WikitTeam/ProjectWikit/internal/modules"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type Repository struct {
	ctx   context.Context
	db    *db.DB
	users *printuser.Renderer
	opts  Options
	forms *formLoader
}

// Options is what the modules read besides the database.
type Options struct {
	Loc  *i18n.Localizer
	Site *db.Site
	User *db.User

	// Render is how a module runs the wikitext it produced. The engine is not
	// reachable from here, so the caller that owns it supplies this.
	Render func(source string, pc *page.Context) (string, error)

	RenderMessage     func(source string) (string, error)
	RenderMessageText func(source string) (string, error)

	Vars page.VarSource

	// ClientIP is what the action log records, and the entry layer is the only
	// layer that knows which forwarded address to believe.
	ClientIP *netip.Addr
}

var _ callbacks.Repository = (*Repository)(nil)

func New(ctx context.Context, d *db.DB, users *printuser.Renderer, opts Options) *Repository {
	return &Repository{ctx: ctx, db: d, users: users, opts: opts, forms: newFormLoader(ctx, d)}
}

func (r *Repository) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	titles, err := r.db.ArticleTitles(r.ctx, r.siteID(), refs)
	if err != nil {
		return nil, err
	}
	out := make([]renderer.PartialPageInfo, 0, len(titles))
	for _, ref := range refs {
		title, ok := titles[ref]
		if !ok {
			// fetch_internal_links omits missing pages instead of reporting
			// exists=false; ftml treats an absent entry as a red link.
			continue
		}
		out = append(out, renderer.PartialPageInfo{FullName: ref, Title: &title, Exists: true})
	}
	return out, nil
}

func (r *Repository) IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		if slug, _ := wikidot.SplitSiteRef(ref.FullName); slug == "" {
			names = append(names, ref.FullName)
		}
	}
	sources, err := r.db.ArticleSources(r.ctx, r.siteID(), names)
	if err != nil {
		return nil, err
	}
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		page := renderer.FetchedPage{FullName: ref.FullName}
		slug, name := wikidot.SplitSiteRef(ref.FullName)
		if slug == "" {
			if source, ok := sources[ref.FullName]; ok {
				page.Content = &source
			}
			out = append(out, page)
			continue
		}
		source, err := r.offSiteSource(slug, name)
		if err != nil {
			return nil, err
		}
		page.Content = source
		out = append(out, page)
	}
	return out, nil
}

// Reading another wiki goes through that wiki's own roles, so a reader who is
// nobody there sees only what its anonymous visitor sees.
func (r *Repository) offSiteSource(slug, name string) (*string, error) {
	other, err := r.db.SiteBySlug(r.ctx, slug)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	article, err := r.db.ArticleByName(r.ctx, other.ID, name)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	resolver := NewPermsOn(r.ctx, r.db, other)
	subject, err := resolver.Subject(r.opts.User, time.Now())
	if err != nil {
		return nil, err
	}
	object, err := resolver.Article(article, r.opts.User)
	if err != nil {
		return nil, err
	}
	if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
		return nil, nil
	}

	sources, err := r.db.ArticleSources(r.ctx, other.ID, []string{name})
	if err != nil {
		return nil, err
	}
	if source, ok := sources[name]; ok {
		return &source, nil
	}
	return nil, nil
}

func (r *Repository) moduleEnv(pc *page.Context) module.Env {
	return module.Env{
		Page:          pc,
		Loc:           r.opts.Loc,
		Site:          r.opts.Site,
		User:          r.opts.User,
		Data:          moduleData{repo: r},
		Render:        r.opts.Render,
		RenderMessage: r.opts.RenderMessage,

		RenderMessageText: r.opts.RenderMessageText,
		Vars:              r.opts.Vars,
	}
}

// CallAPI answers the page asking a module a question of its own, which is not
// a render and so produces JSON rather than markup.
func (r *Repository) CallAPI(pc *page.Context, name, method string, params map[string]string) (wikijson.Object, error) {
	fn, _, ok := module.LookupAPI(name, method)
	if !ok {
		return nil, fmt.Errorf("repo: %s has no api method %s", name, method)
	}
	env := r.moduleEnv(pc)
	env.Name = name
	out, err := fn(env, params)
	var moduleErr *module.Error
	if errors.As(err, &moduleErr) {
		return nil, &callbacks.ModuleError{Message: moduleErr.Message}
	}
	return out, err
}

func (r *Repository) RenderModule(pc *page.Context, name string, params map[string]string, body string) (string, error) {
	html, err := module.Render(r.moduleEnv(pc), name, params, body)
	var moduleErr *module.Error
	if errors.As(err, &moduleErr) {
		return "", &callbacks.ModuleError{Message: moduleErr.Message}
	}
	return html, err
}

// The external: prefix never touches the database, wd: may only ever match an
// imported account, and a bare name is matched against both name columns.
func (r *Repository) RenderUser(username string, avatar bool) (string, error) {
	opts := printuser.Options{Avatar: avatar, Hover: true}

	if external, ok := cutPrefixFold(username, "external:"); ok {
		return r.users.External(external, opts), nil
	}

	var (
		user *db.User
		err  error
	)
	if wd, ok := cutPrefixFold(username, "wd:"); ok {
		user, err = r.db.UserByWikidotName(r.ctx, wd)
	} else {
		user, err = r.db.UserByName(r.ctx, wikidot.CanonicalizeUsername(username))
		// An imported account keeps the name it had on the other site, which
		// need not fold into the name it is shown under here.
		if errors.Is(err, db.ErrNotFound) {
			user, err = r.db.UserByDisplayName(r.ctx, username)
		}
	}
	if errors.Is(err, db.ErrNotFound) {
		return "", callbacks.ErrUserNotFound
	}
	if err != nil {
		return "", err
	}

	return r.renderUser(user, opts)
}

func (r *Repository) renderUser(user *db.User, opts printuser.Options) (string, error) {
	roleList, err := r.db.RolesByUser(r.ctx, r.siteID(), user.ID)
	if err != nil {
		return "", err
	}
	return r.users.User(printuser.User{
		ID:              user.ID,
		Type:            user.Type,
		Username:        user.Username,
		WikidotUsername: user.WikidotUsername,
		DisplayName:     user.DisplayName,
		Avatar:          user.Avatar,
		IsActive:        user.ActiveAt(time.Now()),
	}, roleList, opts)
}

func cutPrefixFold(s, prefix string) (string, bool) {
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return "", false
	}
	return s[len(prefix):], true
}

func (r *Repository) siteID() int64 {
	if r.opts.Site == nil {
		return 0
	}
	return r.opts.Site.ID
}
