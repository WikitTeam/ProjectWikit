// Package localitem answers the three URLs that hand out one piece of a page's
// own source rather than a file.
package localitem

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	CodePrefix  = "/local--code/"
	HTMLPrefix  = "/local--html/"
	ThemePrefix = "/local--theme/"
)

const (
	htmlMime = "text/html; charset=utf-8"
	cssMime  = "text/css; charset=utf-8"
	textMime = "text/plain; charset=utf-8"

	allowedMethods = "GET, HEAD, OPTIONS"

	noArticle  = "Article not found"
	noPerm     = "Permission denied"
	noCode     = "Code block not found"
	noHTML     = "HTML block not found"
	noResource = "Not found"
)

type Deps struct {
	DB     *db.DB
	Engine renderer.Renderer
	Bundle *i18n.Bundle
	Icons  roles.IconLoader
	Log    *slog.Logger

	Now func() time.Time
}

type handler struct {
	deps   Deps
	prefix string
	answer func(req *request, rest string) (item, error)
}

var _ http.Handler = (*handler)(nil)

func NewCode(d Deps) http.Handler {
	h := &handler{deps: withDefaults(d), prefix: CodePrefix}
	h.answer = h.code
	return h
}

func NewHTML(d Deps) http.Handler {
	h := &handler{deps: withDefaults(d), prefix: HTMLPrefix}
	h.answer = h.html
	return h
}

func NewTheme(d Deps) http.Handler {
	h := &handler{deps: withDefaults(d), prefix: ThemePrefix}
	h.answer = h.theme
	return h
}

func withDefaults(d Deps) Deps {
	if d.Now == nil {
		d.Now = time.Now
	}
	return d
}

func (h *handler) log() *slog.Logger {
	if h.deps.Log == nil {
		return slog.Default()
	}
	return h.deps.Log
}

func (h *handler) now() time.Time { return h.deps.Now() }

type item struct {
	status      int
	contentType string
	body        string
}

func found(contentType, body string) item {
	return item{status: http.StatusOK, contentType: contentType, body: body}
}

func missing(body string) item {
	return item{status: http.StatusNotFound, contentType: htmlMime, body: body}
}

type request struct {
	env     *pagerender.Env
	article *db.Article
	user    *db.User
	source  string
	params  page.PathParams
	query   url.Values
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", allowedMethods)
		w.Header().Set("Content-Type", htmlMime)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", allowedMethods)
		write(w, r, item{status: http.StatusMethodNotAllowed, contentType: textMime,
			body: http.StatusText(http.StatusMethodNotAllowed)})
		return
	}

	name, rest, ok := split(r.URL.Path, h.prefix)
	if !ok {
		write(w, r, missing(noResource))
		return
	}

	req, out, err := h.load(r, name)
	if err != nil {
		h.serverError(w, r, err)
		return
	}
	if req == nil {
		write(w, r, out)
		return
	}

	out, err = h.answer(req, rest)
	if err != nil {
		h.serverError(w, r, err)
		return
	}
	write(w, r, out)
}

func (h *handler) serverError(w http.ResponseWriter, r *http.Request, err error) {
	h.log().Error("render local item", "path", r.URL.Path, "err", err)
	write(w, r, item{status: http.StatusInternalServerError, contentType: textMime,
		body: http.StatusText(http.StatusInternalServerError)})
}

func write(w http.ResponseWriter, r *http.Request, out item) {
	w.Header().Set("Content-Type", out.contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(out.body)))
	w.WriteHeader(out.status)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := io.WriteString(w, out.body); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		return
	}
}

func split(path, prefix string) (name, rest string, ok bool) {
	tail, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return "", "", false
	}
	name, rest, ok = strings.Cut(tail, "/")
	if !ok || name == "" || rest == "" || strings.Contains(rest, "/") {
		return "", "", false
	}
	return name, rest, true
}

func (h *handler) load(r *http.Request, name string) (*request, item, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return nil, item{}, errors.New("localitem: the request carries no site")
	}
	user := auth.FromContext(ctx)

	article, err := h.deps.DB.ArticleByName(ctx, name)
	if errors.Is(err, db.ErrNotFound) {
		return nil, missing(noArticle), nil
	}
	if err != nil {
		return nil, item{}, err
	}

	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, h.now())
	if err != nil {
		return nil, item{}, err
	}
	object, err := perm.Article(article, user)
	if err != nil {
		return nil, item{}, err
	}
	if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
		return nil, item{status: http.StatusForbidden, contentType: htmlMime, body: noPerm}, nil
	}

	query := r.URL.Query()
	source, err := h.source(ctx, article, query.Get("revNum"))
	if err != nil {
		return nil, item{}, err
	}

	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(ctx, loc, current, user)

	return &request{
		env:     env,
		article: article,
		user:    user,
		source:  source,
		params:  page.ParsePathParams(query.Get("pathParams")),
		query:   query,
	}, item{}, nil
}

// A revision that is not a number reads as none asked for, so the reader gets
// the current source rather than an error.
func (h *handler) source(ctx context.Context, article *db.Article, revNum string) (string, error) {
	number, err := strconv.Atoi(revNum)
	if err != nil {
		source, err := h.deps.DB.LatestSource(ctx, article.ID)
		if errors.Is(err, db.ErrNotFound) {
			return "", nil
		}
		return source, err
	}
	source, err := h.deps.DB.SourceAtRevision(ctx, article.ID, number)
	if errors.Is(err, db.ErrNotFound) {
		return "", nil
	}
	return source, err
}

func stringMap(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
