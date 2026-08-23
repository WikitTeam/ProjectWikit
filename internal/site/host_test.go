package site

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type fakeLookup struct {
	site *db.Site
	err  error
	got  []string
}

func (f *fakeLookup) SiteByHosts(_ context.Context, hosts []string) (*db.Site, error) {
	f.got = hosts
	return f.site, f.err
}

func marker(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Handler", name)
		w.WriteHeader(http.StatusOK)
	})
}

func newRules(t *testing.T, lookup *fakeLookup) *HostRules {
	t.Helper()
	return NewHostRules(lookup, "8080", marker("next"), marker("unresolved"))
}

func serve(h http.Handler, host, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.Host = host
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

var splitSite = &db.Site{Domain: "wiki.example", MediaDomain: "media.example"}

func TestHostRulesRedirectsMediaPathToMediaHost(t *testing.T) {
	w := serve(newRules(t, &fakeLookup{site: splitSite}), "wiki.example", "/local--files/a.png?v=1")

	if w.Code != http.StatusFound {
		t.Errorf("GET /local--files/a.png on wiki.example = %d, want %d", w.Code, http.StatusFound)
	}
	if got, want := w.Header().Get("Location"), "//media.example/local--files/a.png?v=1"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
	if got := w.Body.Len(); got != 0 {
		t.Errorf("body length = %d, want 0", got)
	}
}

func TestHostRulesRedirectCarriesNoHeaders(t *testing.T) {
	w := serve(newRules(t, &fakeLookup{site: splitSite}), "wiki.example", "/local--files/a.png")

	for _, name := range []string{"Access-Control-Allow-Origin", "X-Content-Type-Options", "X-Frame-Options"} {
		if got := w.Header().Get(name); got != "" {
			t.Errorf("%s = %q, want %q", name, got, "")
		}
	}
}

func TestHostRulesServesOnMediaHost(t *testing.T) {
	w := serve(newRules(t, &fakeLookup{site: splitSite}), "media.example", "/local--files/a.png")

	if got, want := w.Header().Get("X-Handler"), "next"; got != want {
		t.Errorf("X-Handler = %q, want %q", got, want)
	}
	if got, want := w.Header().Get("Access-Control-Allow-Origin"), "*"; got != want {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, want)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "")
	}
}

func TestHostRulesUnknownHostGoesUpstream(t *testing.T) {
	w := serve(newRules(t, &fakeLookup{err: db.ErrNotFound}), "other.example", "/local--files/a.png")

	if got, want := w.Header().Get("X-Handler"), "unresolved"; got != want {
		t.Errorf("X-Handler = %q, want %q", got, want)
	}
}

func TestHostRulesLookupErrorIsNotAnUnresolvedHost(t *testing.T) {
	w := serve(newRules(t, &fakeLookup{err: errors.New("connection refused")}), "wiki.example", "/local--files/a.png")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET with a failing lookup = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if got := w.Header().Get("X-Handler"); got != "" {
		t.Errorf("X-Handler = %q, want %q", got, "")
	}
}

func TestHostRulesPassesBothHostCandidates(t *testing.T) {
	lookup := &fakeLookup{site: splitSite}
	serve(newRules(t, lookup), "media.example", "/local--files/a.png")

	want := []string{"media.example:8080", "media.example"}
	if len(lookup.got) != len(want) {
		t.Fatalf("SiteByHosts got %v, want %v", lookup.got, want)
	}
	for i := range want {
		if lookup.got[i] != want[i] {
			t.Errorf("SiteByHosts got[%d] = %q, want %q", i, lookup.got[i], want[i])
		}
	}
}
