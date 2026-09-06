// Package admin answers the pages a site is administered on.
package admin

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/pageconfig"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

const (
	Prefix = "/-/admin/"
	Bare   = "/-/admin"
)

//go:embed templates/*.html
var files embed.FS

type Deps struct {
	DB       *db.DB
	Bundle   *i18n.Bundle
	Assets   *static.Assets
	Files    string
	TimeZone *time.Location
	Log      *slog.Logger
}

func (d Deps) logger() *slog.Logger {
	if d.Log == nil {
		return slog.Default()
	}
	return d.Log
}

type screen struct {
	slug  string
	label string
	need  string
	serve func(*Handler, http.ResponseWriter, *http.Request, *i18n.Localizer) error
}

var screens []screen

func register(s screen) { screens = append(screens, s) }

type Handler struct {
	deps      Deps
	upstream  http.Handler
	templates *template.Template
}

var _ http.Handler = (*Handler)(nil)

func New(d Deps, upstream http.Handler) (*Handler, error) {
	t, err := template.New("admin").Funcs(funcs()).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{deps: d, upstream: upstream, templates: t}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == Bare {
		seeOther(w, Prefix, http.StatusMovedPermanently)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, Prefix)
	if !ok {
		h.upstream.ServeHTTP(w, r)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if to, ownLogin := siteSignIn(r.URL.Path); ownLogin {
		seeOther(w, to, http.StatusFound)
		return
	}

	granted, staff, err := h.access(ctx)
	if err != nil {
		h.deps.logger().Error("resolve admin access", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	if !staff {
		if auth.FromContext(ctx) == nil {
			seeOther(w, account.LoginPath+"?to="+url.QueryEscape(r.URL.Path), http.StatusFound)
			return
		}
		h.finish(w, r, loc, h.forbidden(w, r, loc))
		return
	}

	head, _, _ := strings.Cut(rest, "/")
	if head == "" {
		h.finish(w, r, loc, h.index(w, r, loc, granted))
		return
	}
	for _, s := range screens {
		if s.slug != head {
			continue
		}
		if !granted.Has(s.need) {
			h.upstream.ServeHTTP(w, r)
			return
		}
		h.finish(w, r, loc, s.serve(h, w, r, loc))
		return
	}
	h.upstream.ServeHTTP(w, r)
}

func siteSignIn(path string) (string, bool) {
	switch {
	case strings.HasPrefix(path, Prefix+"login"):
		return account.LoginPath + "?to=" + url.QueryEscape(Prefix), true
	case strings.HasPrefix(path, Prefix+"logout"):
		return account.LogoutPath, true
	}
	return "", false
}

func (h *Handler) forbidden(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	current := site.FromContext(r.Context())
	icon := ""
	if current.AuthIcon != "" {
		icon = "/local--files/" + current.AuthIcon
	}
	body, err := shell.New(loc, h.deps.Assets, h.deps.TimeZone).Notice(shell.Notice{
		AuthIcon:  icon,
		SiteTitle: current.Title,
		Heading:   loc.T("admin.denied-title"),
		Body:      loc.T("admin.denied"),
		LinkURL:   "/",
		LinkText:  loc.T("system.back-home"),
	})
	if err != nil {
		return err
	}
	return h.write(w, r, loc, loc.T("admin.denied-title"), body, http.StatusForbidden)
}

func (h *Handler) finish(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, err error) {
	if err == nil {
		return
	}
	h.deps.logger().Error("serve admin", "path", r.URL.Path, "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *Handler) access(ctx context.Context) (perms.Set, bool, error) {
	user := auth.FromContext(ctx)
	if user == nil {
		return perms.Set{}, false, nil
	}
	own, err := h.deps.DB.RolesByUser(ctx, user.ID)
	if err != nil {
		return perms.Set{}, false, err
	}
	if !pageconfig.IsStaff(user, own) {
		return perms.Set{}, false, nil
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(user, time.Now())
	if err != nil {
		return perms.Set{}, false, err
	}
	return perms.Resolve(subject, nil), true, nil
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, granted perms.Set) error {
	type entry struct {
		Href  string
		Label string
	}
	list := make([]entry, 0, len(screens))
	for _, s := range screens {
		if !granted.Has(s.need) {
			continue
		}
		list = append(list, entry{Href: Prefix + s.slug + "/", Label: loc.T(s.label)})
	}
	return h.page(w, r, loc, loc.T("admin.title"), "index.html", map[string]any{"Screens": list})
}

func (h *Handler) page(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, title, name string, data any) error {
	var body strings.Builder
	if err := h.templates.ExecuteTemplate(&body, name, data); err != nil {
		return err
	}
	return h.write(w, r, loc, title, body.String(), http.StatusOK)
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, title, body string, status int) error {
	current := site.FromContext(r.Context())
	theme, err := site.ThemeURLByID(r.Context(), h.deps.DB, current.SystemThemeID)
	if err != nil {
		return err
	}
	var out strings.Builder
	err = shell.New(loc, h.deps.Assets, h.deps.TimeZone).SystemPage(&out, shell.System{
		Title:     title,
		SiteTitle: current.Title,
		ThemeURL:  theme,
		BodyClass: "wikit-page admin",
		Heading:   title,
		Content:   body,
	})
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(out.String()))
	}
	return nil
}

func authIcon(s *db.Site) string {
	if s == nil || s.AuthIcon == "" {
		return ""
	}
	return "/local--files/" + s.AuthIcon
}

func funcs() template.FuncMap {
	return template.FuncMap{
		"esc": escape.HTML,
		"dict": func(pairs ...any) map[string]any {
			out := make(map[string]any, len(pairs)/2)
			for i := 0; i+1 < len(pairs); i += 2 {
				out[pairs[i].(string)] = pairs[i+1]
			}
			return out
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"has": func(list []int64, want int64) bool { return slices.Contains(list, want) },
		"stamp": func(at *time.Time) string {
			if at == nil {
				return ""
			}
			return at.Format("2006-01-02T15:04")
		},
		"num": func(p *int) string {
			if p == nil {
				return ""
			}
			return strconv.Itoa(*p)
		},
		"deref": func(p *int64) int64 {
			if p == nil {
				return 0
			}
			return *p
		},
	}
}
