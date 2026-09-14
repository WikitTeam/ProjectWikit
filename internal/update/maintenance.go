package update

import (
	"net/http"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

func MaintenanceHandler(bundle *i18n.Bundle, assets *static.Assets, files http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, static.Prefix) && files != nil {
			files.ServeHTTP(w, r)
			return
		}
		loc := bundle.Localizer(bundle.Match(r.Header.Get(i18n.AcceptHeader)))
		render := shell.New(loc, assets)
		body, err := render.Notice(shell.Notice{
			Heading: loc.T("update.maintenance-title"),
			Body:    loc.T("update.maintenance"),
		})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		var out strings.Builder
		if err := render.SystemPage(&out, shell.System{
			Title:     loc.T("update.maintenance-title"),
			BodyClass: "wikit-page",
			Content:   body,
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusServiceUnavailable)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte(out.String()))
		}
	})
}
