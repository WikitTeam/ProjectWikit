package lang

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

func bundle(t *testing.T) *i18n.Bundle {
	t.Helper()
	dir := t.TempDir()
	raw, err := json.Marshal(map[string]string{"toc-open": "Expand"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := i18n.Load(dir)
	if err != nil {
		t.Fatalf("Load() err = %v, want nil", err)
	}
	return b
}

func serve(t *testing.T, accept string, current *db.Site, user *db.User) (string, http.Header) {
	t.Helper()
	var chosen string
	handler := Middleware(bundle(t))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chosen = i18n.LanguageFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if accept != "" {
		req.Header.Set(i18n.AcceptHeader, accept)
	}
	ctx := req.Context()
	if current != nil {
		ctx = site.WithSite(ctx, current)
	}
	if user != nil {
		ctx = auth.NewContext(ctx, user)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req.WithContext(ctx))
	return chosen, rec.Header()
}

func TestMiddlewarePrefersTheMemberChoice(t *testing.T) {
	got, _ := serve(t, "zh-CN", &db.Site{Language: "zh-hans"}, &db.User{Language: "en"})
	if got != "en" {
		t.Errorf("LanguageFrom(ctx) = %q, want %q", got, "en")
	}
}

func TestMiddlewareFallsBackToTheBrowser(t *testing.T) {
	got, _ := serve(t, "en-GB", &db.Site{Language: "zh-hans"}, &db.User{})
	if got != "en" {
		t.Errorf("LanguageFrom(ctx) = %q, want %q", got, "en")
	}
}

func TestMiddlewareFallsBackToTheSite(t *testing.T) {
	got, _ := serve(t, "de", &db.Site{Language: "en"}, nil)
	if got != "en" {
		t.Errorf("LanguageFrom(ctx) = %q, want %q", got, "en")
	}
}

func TestMiddlewareWithoutASiteOrAUser(t *testing.T) {
	got, _ := serve(t, "", nil, nil)
	if got != i18n.DefaultLanguage {
		t.Errorf("LanguageFrom(ctx) = %q, want %q", got, i18n.DefaultLanguage)
	}
}

func TestMiddlewareVariesOnAcceptLanguage(t *testing.T) {
	_, header := serve(t, "en", nil, nil)
	if got := header.Get("Vary"); got != "Accept-Language" {
		t.Errorf("Vary = %q, want %q", got, "Accept-Language")
	}
}
