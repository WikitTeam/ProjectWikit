// Package userpage answers the profile a user chip links to.
package userpage

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const Prefix = "/-/users/"

const notFoundBody = "User not found"

type Deps struct {
	DB       *db.DB
	Engine   renderer.Renderer
	Bundle   *i18n.Bundle
	Icons    roles.IconLoader
	Assets   *static.Assets
	TimeZone *time.Location
	Log      *slog.Logger
}

type Handler struct {
	deps Deps
}

var _ http.Handler = (*Handler)(nil)

func New(d Deps) *Handler {
	if d.TimeZone == nil {
		d.TimeZone = time.UTC
	}
	return &Handler{deps: d}
}

func (h *Handler) log() *slog.Logger {
	if h.deps.Log == nil {
		return slog.Default()
	}
	return h.deps.Log
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	name, ok := strings.CutPrefix(r.URL.Path, Prefix)
	if !ok || name == "" || strings.Contains(name, "/") {
		notFound(w)
		return
	}

	body, err := h.page(r, name)
	if err != nil {
		h.log().Error("render profile", "path", r.URL.Path, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if body == "" {
		notFound(w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
}

func (h *Handler) page(r *http.Request, name string) (string, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", errors.New("userpage: the request carries no site")
	}
	viewer := auth.FromContext(ctx)
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)

	profile, err := h.lookup(r, name)
	if errors.Is(err, db.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	data, err := h.data(r, loc, current, profile, viewer)
	if err != nil {
		return "", err
	}
	render := shell.New(loc, h.deps.Assets, h.deps.TimeZone)
	content, err := render.Profile(data)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	err = render.SystemPage(&out, shell.System{
		Title:     data.DisplayName,
		BodyClass: "wikit-page",
		Content:   content,
	})
	return out.String(), err
}

func (h *Handler) lookup(r *http.Request, name string) (*db.Profile, error) {
	if id, ok := numericID(name); ok {
		return h.deps.DB.ProfileByID(r.Context(), id)
	}
	return h.deps.DB.ProfileByName(r.Context(), wikidot.CanonicalizeUsername(name))
}

// The tail after the id is never looked up, but it has to be there and it has
// to be a slug, or the whole thing is read as a name instead.
func numericID(name string) (int64, bool) {
	digits, slug, found := strings.Cut(name, "-")
	if !found || slug == "" || !isSlug(slug) {
		return 0, false
	}
	id, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || id < 0 {
		return 0, false
	}
	return id, true
}

func isSlug(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func (h *Handler) data(r *http.Request, loc *i18n.Localizer, current *db.Site,
	profile *db.Profile, viewer *db.User) (shell.Profile, error) {

	ctx := r.Context()
	data := shell.Profile{
		ID:          profile.ID,
		DisplayName: displayName(profile),
		Avatar:      avatar(profile),
		AuthIcon:    authIcon(current),
		FromWikidot: profile.Type == printuser.TypeWikidot,
		IsBot:       profile.Type == printuser.TypeBot,
		FullName:    strings.TrimSpace(profile.FirstName + " " + profile.LastName),
		Bio:         profile.Bio,
		JoinedAt:    profile.DateJoined,
	}

	titles, err := h.titles(ctx, loc, profile)
	if err != nil {
		return shell.Profile{}, err
	}
	data.Subtitle = strings.Join(titles, ", ")

	if viewer != nil && viewer.ID != profile.ID {
		blocked, err := h.deps.DB.DirectMessageBlocked(ctx, viewer.ID, profile.ID)
		if err != nil {
			return shell.Profile{}, err
		}
		data.CanDirectMessage = true
		data.IsBlocked = blocked
		config, err := wikijson.Marshal(wikijson.Object{
			{Key: "userId", Value: profile.ID},
			{Key: "isBlocked", Value: blocked},
		})
		if err != nil {
			return shell.Profile{}, err
		}
		data.ActionsConfig = config
	}
	data.IsSelf = viewer != nil && viewer.ID == profile.ID

	if profile.Bio != "" {
		html, err := h.bio(r, loc, current, profile, viewer)
		if err != nil {
			return shell.Profile{}, err
		}
		data.BioHTML = html
	}
	return data, nil
}

// The two states that outrank every role are answered first, the way the chip
// beside a name answers them.
func (h *Handler) titles(ctx context.Context, loc *i18n.Localizer, profile *db.Profile) ([]string, error) {
	switch {
	case !profile.IsActive && profile.Type == printuser.TypeWikidot:
		return []string{loc.T("user-inactive-title")}, nil
	case !profile.IsActive:
		return []string{loc.T("user-banned-title")}, nil
	case profile.Type == printuser.TypeBot:
		return []string{loc.T("user-bot-title")}, nil
	}
	rs, err := h.deps.DB.RolesByUser(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	return roles.ShowcaseOf(rs).Titles, nil
}

func (h *Handler) bio(r *http.Request, loc *i18n.Localizer, current *db.Site,
	profile *db.Profile, viewer *db.User) (string, error) {

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(r.Context(), loc, current, viewer)
	pc := page.NewContext(nil, nil, nil, viewer)
	info, err := env.PageInfo(nil)
	if err != nil {
		return "", err
	}
	html, err := env.HTML(profile.Bio, info, env.Callbacks(nil, pc), renderer.ModeInline)
	if err != nil {
		return "", err
	}
	return html.Body, nil
}

func displayName(p *db.Profile) string {
	if p.Type == printuser.TypeWikidot {
		return "wd:" + firstNonEmpty(p.DisplayName, p.WikidotUsername)
	}
	return firstNonEmpty(p.DisplayName, p.Username)
}

func authIcon(s *db.Site) string {
	if s.AuthIcon == "" {
		return ""
	}
	return "/local--files/" + s.AuthIcon
}

func avatar(p *db.Profile) string {
	if p.Type == printuser.TypeWikidot {
		return printuser.WikidotAvatar
	}
	if p.Avatar == "" {
		return printuser.DefaultAvatar
	}
	return "/local--files/" + p.Avatar
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(notFoundBody)))
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(notFoundBody))
}
