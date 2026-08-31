// Package repo composes the data layer into the renderer's Repository.
package repo

import (
	"context"
	"errors"
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
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

type Repository struct {
	ctx   context.Context
	db    *db.DB
	users *printuser.Renderer
	opts  Options
}

// Options is what the modules read besides the database.
type Options struct {
	Loc  *i18n.Localizer
	Site *db.Site
	User *db.User

	// Render is how a module runs the wikitext it produced. The engine is not
	// reachable from here, so the caller that owns it supplies this.
	Render func(source string, pc *page.Context) (string, error)

	RenderMessage func(source string) (string, error)

	Vars page.VarSource
}

var _ callbacks.Repository = (*Repository)(nil)

func New(ctx context.Context, d *db.DB, users *printuser.Renderer, opts Options) *Repository {
	return &Repository{ctx: ctx, db: d, users: users, opts: opts}
}

func (r *Repository) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	titles, err := r.db.ArticleTitles(r.ctx, refs)
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
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.FullName
	}
	sources, err := r.db.ArticleSources(r.ctx, names)
	if err != nil {
		return nil, err
	}
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		page := renderer.FetchedPage{FullName: ref.FullName}
		if source, ok := sources[ref.FullName]; ok {
			page.Content = &source
		}
		out = append(out, page)
	}
	return out, nil
}

func (r *Repository) RenderModule(pc *page.Context, name string, params map[string]string, body string) (string, error) {
	html, err := module.Render(module.Env{
		Page:          pc,
		Loc:           r.opts.Loc,
		Site:          r.opts.Site,
		User:          r.opts.User,
		Data:          moduleData{repo: r},
		Render:        r.opts.Render,
		RenderMessage: r.opts.RenderMessage,
		Vars:          r.opts.Vars,
	}, name, params, body)
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
		user, err = r.db.UserByWikidotName(r.ctx, wikidot.CanonicalizeUsername(wd))
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
	roleList, err := r.db.RolesByUser(r.ctx, user.ID)
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
