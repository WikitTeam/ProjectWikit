package site

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestThemeURLWithoutTheme(t *testing.T) {
	if got := ThemeURL(nil); got != "" {
		t.Errorf("ThemeURL(nil) = %q, want %q", got, "")
	}
}

func TestThemeURLOfInlineTheme(t *testing.T) {
	theme := &db.Theme{Slug: "dark", Mode: db.ThemeInline, UpdatedAt: time.Unix(1700000000, 0)}
	want := "/-/theme/dark.css?v=1700000000"
	if got := ThemeURL(theme); got != want {
		t.Errorf("ThemeURL(inline) = %q, want %q", got, want)
	}
}

func TestThemeURLOfExternalTheme(t *testing.T) {
	theme := &db.Theme{Slug: "dark", Mode: db.ThemeExternal, ExternalURL: "  https://cdn.example/x.css  "}
	want := "https://cdn.example/x.css"
	if got := ThemeURL(theme); got != want {
		t.Errorf("ThemeURL(external) = %q, want %q", got, want)
	}
}

func TestThemeURLOfExternalThemeWithoutURL(t *testing.T) {
	theme := &db.Theme{Slug: "dark", Mode: db.ThemeExternal, ExternalURL: "   "}
	if got := ThemeURL(theme); got != "" {
		t.Errorf("ThemeURL(external without url) = %q, want %q", got, "")
	}
}

func themeRequest(t *testing.T, dir, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	NewThemeFiles(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func themeDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "theme"), 0o755); err != nil {
		t.Fatalf("MkdirAll() err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.Join(root, "theme", "night.css"), []byte("body{color:#fff}"), 0o644); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
	return root
}

func TestThemeFilesServesAStylesheet(t *testing.T) {
	rec := themeRequest(t, themeDir(t), "/-/theme/night.css")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET night.css = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "body{color:#fff}" {
		t.Errorf("GET night.css body = %q, want %q", got, "body{color:#fff}")
	}
	if got := rec.Header().Get("Content-Type"); got != themeMime {
		t.Errorf("GET night.css content-type = %q, want %q", got, themeMime)
	}
	if got := rec.Header().Get("Cache-Control"); got != themeCache {
		t.Errorf("GET night.css cache-control = %q, want %q", got, themeCache)
	}
}

func TestThemeFilesAnswers404(t *testing.T) {
	root := themeDir(t)
	for _, target := range []string{
		"/-/theme/nosuchtheme.css",
		"/-/theme/night",
		"/-/theme/.css",
		"/-/theme/../../manage.py.css",
		"/-/theme/",
	} {
		if got := themeRequest(t, root, target).Code; got != http.StatusNotFound {
			t.Errorf("GET %s = %d, want %d", target, got, http.StatusNotFound)
		}
	}
}

func TestThemeSlugDropsWhatItCannotName(t *testing.T) {
	cases := map[string]string{
		"night":     "night",
		"night-2_a": "night-2_a",
		"../../etc": "etc",
		"nig ht":    "night",
		"nig/ht":    "night",
		"":          "",
	}
	for in, want := range cases {
		if got := themeSlug(in); got != want {
			t.Errorf("themeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}
