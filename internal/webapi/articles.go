package webapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const ArticlesPrefix = "/pw-api/articles/"

const (
	tailLinks   = "links"
	tailVersion = "version"
	tailVotes   = "votes"
	tailLog     = "log"
)

type Articles struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Articles)(nil)

func NewArticles(d Deps, upstream http.Handler) *Articles {
	return &Articles{deps: d, upstream: upstream}
}

// Only the tails listed here are answered. A page's own URL still belongs to
// the upstream, which is where saving lives.
func (h *Articles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	if r.URL.Path == CreatePath && r.Method == http.MethodPost {
		h.create(w, r, loc)
		return
	}

	name, tail, found := cutLast(strings.TrimPrefix(r.URL.Path, ArticlesPrefix))
	if !found {
		h.page(w, r, loc, strings.TrimPrefix(r.URL.Path, ArticlesPrefix))
		return
	}
	if name == "" {
		h.upstream.ServeHTTP(w, r)
		return
	}

	switch {
	case tail == tailLinks && r.Method == http.MethodGet:
		h.answer(w, r, loc, name, h.links)
	case tail == tailVersion && r.Method == http.MethodGet:
		h.answer(w, r, loc, name, h.version)
	case tail == tailLog && r.Method == http.MethodGet:
		h.answer(w, r, loc, name, h.log)
	case tail == tailVotes && r.Method == http.MethodGet:
		h.answer(w, r, loc, name, h.votes)
	case tail == tailVotes && r.Method == http.MethodDelete:
		h.answer(w, r, loc, name, h.resetVotes)
	default:
		h.upstream.ServeHTTP(w, r)
	}
}

// The page's own path answers only what it writes. Reading one is still the
// upstream's, which is where the editor asks for it.
func (h *Articles) page(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, name string) {
	switch {
	case name == "":
		h.upstream.ServeHTTP(w, r)
	case r.Method == http.MethodPut:
		h.answer(w, r, loc, name, h.update)
	case r.Method == http.MethodDelete:
		h.answer(w, r, loc, name, h.remove)
	default:
		h.upstream.ServeHTTP(w, r)
	}
}

type answerFunc func(r *http.Request, loc *i18n.Localizer, name string) (string, int, error)

func (h *Articles) answer(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, name string, fn answerFunc) {
	if err := h.viewable(r, name); err != nil {
		h.fail(w, loc, err)
		return
	}
	body, status, err := fn(r, loc, name)
	if err != nil {
		h.fail(w, loc, err)
		return
	}
	writeJSON(w, status, body)
}

var (
	errForbidden = errors.New("webapi: the visitor may not see this page")
	errNotFound  = errors.New("webapi: no such page")
)

func (h *Articles) fail(w http.ResponseWriter, loc *i18n.Localizer, err error) {
	switch {
	case errors.Is(err, errForbidden):
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
	case errors.Is(err, errNotFound):
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
	default:
		h.deps.log().Error("article api", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
	}
}

// The category is asked rather than the page, so a name with no page behind it
// is still refused to whoever may not read that category at all.
func (h *Articles) viewable(r *http.Request, name string) error {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	perm := repo.NewPerms(ctx, h.deps.DB)

	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return err
	}
	category, _ := wikidot.Split(name)
	object, err := perm.Category(category)
	if err != nil {
		return err
	}
	if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
		return errForbidden
	}
	return nil
}

func (h *Articles) article(r *http.Request, name string) (*db.Article, error) {
	found, err := h.deps.DB.ArticleByName(r.Context(), name)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errNotFound
	}
	return found, err
}

func (h *Articles) links(r *http.Request, _ *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}

	children, err := h.deps.DB.ArticleChildren(ctx, article.ID)
	if err != nil {
		return "", 0, err
	}
	kids := make(wikijson.Array, 0, len(children))
	for i := range children {
		kids = append(kids, linkRecord(children[i].FullName(), children[i].Title, true))
	}

	links, err := h.deps.DB.LinksTo(ctx, name)
	if err != nil {
		return "", 0, err
	}
	refs := make([]string, 0, len(links))
	for _, link := range links {
		refs = append(refs, strings.ToLower(link.From))
	}
	titles, err := h.deps.DB.ArticleTitles(ctx, refs)
	if err != nil {
		return "", 0, err
	}

	includes, plain := wikijson.Array{}, wikijson.Array{}
	for _, link := range links {
		from := strings.ToLower(link.From)
		record := linkRecord(from, from, false)
		if title, ok := titles[from]; ok {
			record = linkRecord(from, title, true)
		}
		if link.Type == db.LinkInclude {
			includes = append(includes, record)
		} else if link.Type == db.LinkPlain {
			plain = append(plain, record)
		}
	}

	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "children", Value: kids},
		{Key: "includes", Value: includes},
		{Key: "links", Value: plain},
	})
	return body, http.StatusOK, err
}

func linkRecord(id, title string, exists bool) wikijson.Object {
	return wikijson.Object{
		{Key: "id", Value: id},
		{Key: "title", Value: title},
		{Key: "exists", Value: exists},
	}
}

func (h *Articles) version(r *http.Request, _ *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	revNumber, err := strconv.Atoi(r.URL.Query().Get("revNum"))
	if err != nil {
		return "", 0, errNotFound
	}
	source, err := h.deps.DB.SourceAtRevision(ctx, article.ID, revNumber)
	if errors.Is(err, db.ErrNotFound) || (err == nil && source == "") {
		return "", 0, errNotFound
	}
	if err != nil {
		return "", 0, err
	}

	rendered, err := h.render(r, article, source)
	if err != nil {
		return "", 0, err
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "source", Value: source},
		{Key: "rendered", Value: rendered},
	})
	return body, http.StatusOK, err
}

func (h *Articles) render(r *http.Request, article *db.Article, source string) (string, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", errors.New("webapi: the request carries no site")
	}
	user := auth.FromContext(ctx)
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons, Trust: h.deps.Trust}.
		Env(ctx, loc, current, user)
	env.SetClient(r)

	params := page.ParsePathParams(r.URL.Query().Get("pathParams"))
	vars := env.Vars(article)
	info, err := env.PageInfo(article)
	if err != nil {
		return "", err
	}
	pc := page.NewContext(article, article, params, user)
	html, err := env.HTML(page.PreRender(source, vars), info, env.Callbacks(vars, pc), renderer.ModeArticle)
	if err != nil {
		return "", err
	}
	return html.Body, nil
}

func (h *Articles) votes(r *http.Request, _ *i18n.Localizer, name string) (string, int, error) {
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	body, err := h.moduleAPI(r, article, "rate", "get_votes")
	return body, http.StatusOK, err
}

func (h *Articles) resetVotes(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		return field("error", loc.T("api-csrf-failed")), http.StatusForbidden, nil
	}

	user := auth.FromContext(ctx)
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return "", 0, err
	}
	object, err := perm.Article(article, user)
	if err != nil {
		return "", 0, err
	}
	if !perms.Resolve(subject, object).Has(perms.ResetArticleVotes) {
		return "", 0, errForbidden
	}

	meta, err := h.votesMeta(r, article)
	if err != nil {
		return "", 0, err
	}
	if err := h.deps.DB.DeleteArticleVotes(ctx, article.ID); err != nil {
		return "", 0, err
	}
	var userID *int64
	if user != nil {
		userID = &user.ID
	}
	if _, err := h.deps.DB.AddArticleLogEntry(ctx, article.ID, userID,
		db.LogVotesDeleted, "", meta, time.Now().UTC()); err != nil {
		return "", 0, err
	}
	return field("status", "ok"), http.StatusOK, nil
}

// The entry records what the votes were, since the rows themselves are about to
// be gone.
func (h *Articles) votesMeta(r *http.Request, article *db.Article) (string, error) {
	ctx := r.Context()
	votes, err := h.deps.DB.ArticleVotes(ctx, article.ID)
	if err != nil {
		return "", err
	}
	rows := make(wikijson.Array, 0, len(votes))
	for i := range votes {
		rows = append(rows, wikijson.Object{
			{Key: "user_id", Value: votes[i].User.ID},
			{Key: "vote", Value: votes[i].Rate},
			{Key: "role_id", Value: idOrNil(votes[i].RoleID)},
			{Key: "date", Value: voteDateOrNil(votes[i].Date)},
		})
	}

	rating, err := h.rating(r, article)
	if err != nil {
		return "", err
	}
	return wikijson.Marshal(wikijson.Object{
		{Key: "rating_mode", Value: rating.Mode},
		{Key: "rating", Value: rating.Value},
		{Key: "votes_count", Value: rating.Votes},
		{Key: "popularity", Value: rating.Popularity},
		{Key: "votes", Value: rows},
	})
}

func (h *Articles) moduleAPI(r *http.Request, article *db.Article, name, method string) (string, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", errors.New("webapi: the request carries no site")
	}
	user := auth.FromContext(ctx)
	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons, Trust: h.deps.Trust}.
		Env(ctx, h.deps.Bundle.Localizer(i18n.DefaultLanguage), current, user)
	env.SetClient(r)

	out, err := env.ModuleAPI(page.NewContext(article, article, nil, user), name, method, nil)
	if err != nil {
		return "", err
	}
	return wikijson.Marshal(out)
}

func (h *Articles) rating(r *http.Request, article *db.Article) (page.Rating, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return page.Rating{}, errors.New("webapi: the request carries no site")
	}
	siteMode, err := h.deps.DB.SiteRatingMode(ctx, current.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return page.Rating{}, err
	}
	categoryMode, err := h.deps.DB.CategoryRatingMode(ctx, article.Category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return page.Rating{}, err
	}
	mode := page.RatingMode(siteMode, categoryMode)
	if mode == page.RatingModeDisabled {
		return page.RatingOf(mode, db.VoteStats{}), nil
	}
	stats, err := h.deps.DB.VoteStats(ctx, article.ID)
	if err != nil {
		return page.Rating{}, err
	}
	return page.RatingOf(mode, stats), nil
}

func idOrNil(id *int64) any {
	if id == nil {
		return nil
	}
	return *id
}

func voteDateOrNil(at *time.Time) any {
	if at == nil {
		return nil
	}
	return isoTime(*at)
}

// The name keeps every slash but the last, since a forum page carries some.
func cutLast(path string) (name, tail string, found bool) {
	at := strings.LastIndex(path, "/")
	if at < 0 {
		return "", "", false
	}
	return path[:at], path[at+1:], true
}
