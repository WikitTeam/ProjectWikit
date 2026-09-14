package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func stub(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
}

func table() map[string]http.Handler {
	prefixes := []string{
		"/", "/-/", "/-/admin", "/-/admin/", "/-/login", "/-/signup/", "/-/static/",
		"/pw-api/", "/pw-api/modules", "/pw-api/articles/", "/pw-api/notifications",
		"/local--files/", "/local--theme/",
	}
	out := make(map[string]http.Handler, len(prefixes))
	for _, prefix := range prefixes {
		out[prefix] = stub(prefix)
	}
	return out
}

func TestRouteLongestPrefixWins(t *testing.T) {
	m, err := New(table())
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	tests := map[string]string{
		"/":                              "/",
		"/scp-173":                       "/",
		"/forum:start":                   "/",
		"/-":                             "/",
		"/pw-apidocs":                    "/",
		"/-/admin":                       "/-/admin",
		"/-/admin/":                      "/-/admin/",
		"/-/admin/users/7":               "/-/admin/",
		"/-/preferences/":                "/-/",
		"/-/login":                       "/-/login",
		"/-/signup/check-wikidot":        "/-/signup/",
		"/-/static/app.js":               "/-/static/",
		"/pw-api/articles/scp-173/votes": "/pw-api/articles/",
		"/pw-api/notifications":          "/pw-api/notifications",
		"/pw-api/notify":                 "/pw-api/",
		"/local--files/a/b.png":          "/local--files/",
		"/local--theme/12/style.css":     "/local--theme/",
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := m.Route(path); got != want {
				t.Errorf("Route(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

func TestRouteExactMatchesOnlyTheWholePath(t *testing.T) {
	m, err := New(table())
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	tests := map[string]string{
		"/pw-api/modules":   "/pw-api/modules",
		"/pw-api/modules/":  "/pw-api/",
		"/pw-api/modules/1": "/pw-api/",
		"/pw-api/module":    "/pw-api/",
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := m.Route(path); got != want {
				t.Errorf("Route(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

func TestNewRejectsBadHandlers(t *testing.T) {
	tests := map[string]map[string]http.Handler{
		"no fallback":            {"/pw-api/": stub("api")},
		"prefix without a slash": {"/": stub("root"), "pw-api/": stub("api")},
		"nil handler":            {"/": stub("root"), "/pw-api/": nil},
		"nil fallback":           {"/": nil},
		"empty":                  {},
	}
	for name, handlers := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := New(handlers); err == nil {
				t.Error("New() err = nil, want an error")
			}
		})
	}
}

func TestNewCopiesTheHandlers(t *testing.T) {
	handlers := map[string]http.Handler{"/": stub("root")}
	m, err := New(handlers)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	handlers["/"] = stub("replaced")

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/scp-173", nil))
	if got := rec.Body.String(); got != "root" {
		t.Errorf("ServeHTTP(/scp-173) = %q, want %q", got, "root")
	}
}

func TestServeHTTPDispatches(t *testing.T) {
	m, err := New(map[string]http.Handler{
		"/":               stub("articles"),
		"/pw-api/":        stub("not found"),
		"/pw-api/modules": stub("modules"),
	})
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	tests := map[string]string{
		"/pw-api/modules":  "modules",
		"/pw-api/articles": "not found",
		"/scp-173":         "articles",
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if got := rec.Body.String(); got != want {
				t.Errorf("ServeHTTP(%q) = %q, want %q", path, got, want)
			}
		})
	}
}
