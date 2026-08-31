package site

import (
	"context"
	"errors"
	"net/http"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

const redirectContentType = "text/html; charset=utf-8"

// Lookup resolves a request's Host to a site. *db.DB satisfies it.
type Lookup interface {
	SiteByHosts(ctx context.Context, hosts []string) (*db.Site, error)
}

// HostRules is the redirect between the two domains plus the header pair that
// follows from which one the request arrived on. The asset bundle stays outside
// it, since it is answered before any of this runs.
type HostRules struct {
	sites      Lookup
	serverPort string
	next       http.Handler
	unresolved http.Handler
}

var _ http.Handler = (*HostRules)(nil)

func NewHostRules(sites Lookup, serverPort string, next, unresolved http.Handler) *HostRules {
	return &HostRules{sites: sites, serverPort: serverPort, next: next, unresolved: unresolved}
}

func (h *HostRules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	found, err := h.sites.SiteByHosts(r.Context(), LookupHosts(r.Host, h.serverPort))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			h.unresolved.ServeHTTP(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	decision := Decide(Site{Domain: found.Domain, MediaDomain: found.MediaDomain}, r.Host, r.URL)
	if decision.Action == Redirect {
		// Not http.Redirect, which writes an HTML body for GET where this
		// response has none.
		w.Header().Set("Location", decision.Location)
		w.Header().Set("Content-Type", redirectContentType)
		// Written out because net/http leaves it off a HEAD it did not size.
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusFound)
		return
	}

	for name, value := range decision.Headers {
		w.Header().Set(name, value)
	}
	h.next.ServeHTTP(w, r.WithContext(WithSite(r.Context(), found)))
}
