package site

import (
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

// A wildcard DNS record sends every subdomain to this server, so a mistyped one
// has to land somewhere that says so.
func NewUnresolved(bundle *i18n.Bundle, assets *static.Assets, tz *time.Location) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc := bundle.For(r.Context())
		render := shell.New(loc, assets, tz)
		host := StripPort(r.Host)

		// The host is echoed and nothing else is. Listing the sites that do
		// answer would hand an unconfigured name a directory of them.
		body, err := render.Notice(shell.Notice{
			Heading: loc.T("host.unresolved-title"),
			Body:    loc.T("host.unresolved", "host", host),
		})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		var out strings.Builder
		err = render.SystemPage(&out, shell.System{
			Title:     loc.T("host.unresolved-title"),
			BodyClass: "wikit-page",
			Content:   body,
		})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte(out.String()))
		}
	})
}
