// Package articlepage answers the article URLs, which is every path the wiki
// has not claimed for something else.
package articlepage

import (
	"errors"
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

type Deps struct {
	DB          *db.DB
	Engine      renderer.Renderer
	Bundle      *i18n.Bundle
	Icons       roles.IconLoader
	Assets      *static.Assets
	TimeZone    *time.Location
	GoogleTagID string

	// Now exists so a test can pin the moment an inactive account comes back.
	Now func() time.Time
}

type Handler struct {
	deps  Deps
	shell func(loc *i18n.Localizer) *shell.Renderer
}

const allowedMethods = "GET, HEAD, OPTIONS"

var _ http.Handler = (*Handler)(nil)

func New(d Deps) *Handler {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.TimeZone == nil {
		d.TimeZone = time.UTC
	}
	return &Handler{
		deps: d,
		shell: func(loc *i18n.Localizer) *shell.Renderer {
			return shell.New(loc, d.Assets, d.TimeZone)
		},
	}
}

func (h *Handler) now() time.Time { return h.deps.Now() }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", allowedMethods)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", allowedMethods)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	out, err := h.build(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if out.Location != "" {
		w.Header().Set("Location", out.Location)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(out.Status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(out.Status)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write([]byte(out.Body)); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		return
	}
}
