package mount

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type seen struct {
	path, user, cookie, prefix, spoof string
}

func upstream(t *testing.T) (*httptest.Server, *seen) {
	got := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.user = r.Header.Get(UserHeader)
		got.cookie = r.Header.Get("Cookie")
		got.prefix = r.Header.Get("X-Forwarded-Prefix")
		got.spoof = r.Header.Get("X-Pwikit-Other")
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestNewForwardsSignedReader(t *testing.T) {
	srv, got := upstream(t)
	h, err := New(Config{Path: "/app/", Upstream: srv.URL, Secret: "k"}, quiet())
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/x?y=1", nil)
	req.Header.Set(UserHeader, "forged")
	req.Header.Set("X-Pwikit-Other", "forged")
	req.Header.Set("Cookie", "pwikit_sessionid=s; pwikit_csrftoken=c; keep=1")
	ctx := auth.NewContext(req.Context(), &db.User{ID: 7, Username: "alice"})
	ctx = site.WithSite(ctx, &db.Site{Slug: "demo"})
	h.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))

	if got.path != "/app/x" {
		t.Errorf("path = %q, want /app/x", got.path)
	}
	if got.cookie != "keep=1" {
		t.Errorf("cookie = %q, want keep=1", got.cookie)
	}
	if got.prefix != "/app" {
		t.Errorf("X-Forwarded-Prefix = %q, want /app", got.prefix)
	}
	if got.spoof != "" {
		t.Errorf("X-Pwikit-Other = %q, want empty", got.spoof)
	}
	body, sig, ok := strings.Cut(got.user, ".")
	if !ok || sig != Signature("k", body) {
		t.Fatalf("user header = %q, want a body signed with k", got.user)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(body)
	var id identity
	if err := json.Unmarshal(raw, &id); err != nil {
		t.Fatalf("decode user header err = %v", err)
	}
	if id.ID != 7 || id.Username != "alice" || id.Site != "demo" {
		t.Errorf("identity = %+v, want id 7 alice on demo", id)
	}
	if id.Superuser {
		t.Errorf("identity.Superuser = true, want false")
	}
}

func TestNewMarksSuperuser(t *testing.T) {
	srv, got := upstream(t)
	h, err := New(Config{Path: "/app/", Upstream: srv.URL, Secret: "k"}, quiet())
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	ctx := auth.NewContext(req.Context(), &db.User{ID: 1, Username: "root", IsSuperuser: true})
	h.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))

	body, _, _ := strings.Cut(got.user, ".")
	raw, _ := base64.RawURLEncoding.DecodeString(body)
	var id identity
	if err := json.Unmarshal(raw, &id); err != nil {
		t.Fatalf("decode user header err = %v", err)
	}
	if !id.Superuser {
		t.Errorf("identity.Superuser = false, want true")
	}
}

func TestNewAnonymousHasNoUser(t *testing.T) {
	srv, got := upstream(t)
	h, err := New(Config{Path: "/app/", Upstream: srv.URL, Secret: "k", StripPrefix: true}, quiet())
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/x", nil)
	req.Header.Set(UserHeader, "forged")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got.user != "" {
		t.Errorf("user header = %q, want empty", got.user)
	}
	if got.path != "/x" {
		t.Errorf("path = %q, want /x", got.path)
	}
}

func TestNewRefuses(t *testing.T) {
	cases := map[string]Config{
		"root":     {Path: "/", Upstream: "http://127.0.0.1:1", Secret: "k"},
		"system":   {Path: "/-/x/", Upstream: "http://127.0.0.1:1", Secret: "k"},
		"api":      {Path: "/pw-api/x/", Upstream: "http://127.0.0.1:1", Secret: "k"},
		"slash":    {Path: "/app", Upstream: "http://127.0.0.1:1", Secret: "k"},
		"upstream": {Path: "/app/", Upstream: "ftp://x", Secret: "k"},
		"secret":   {Path: "/app/", Upstream: "http://127.0.0.1:1"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(c, quiet()); err == nil {
				t.Errorf("New(%+v) err = nil, want an error", c)
			}
		})
	}
}
