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

func TestValidateAcceptsTable(t *testing.T) {
	if err := Validate(Table); err != nil {
		t.Errorf("Validate(Table) err = %v, want nil", err)
	}
}

func TestValidateRejectsBadTable(t *testing.T) {
	tests := []struct {
		name  string
		table []Route
	}{
		{"prefix does not start with slash", []Route{{Prefix: "/", Owner: OwnerUpstream}, {Prefix: "api/", Owner: OwnerUpstream}}},
		{"prefix does not end with slash", []Route{{Prefix: "/", Owner: OwnerUpstream}, {Prefix: "/api", Owner: OwnerUpstream}}},
		{"duplicate prefix", []Route{{Prefix: "/", Owner: OwnerUpstream}, {Prefix: "/pw-api/", Owner: OwnerUpstream}, {Prefix: "/pw-api/", Owner: OwnerGo}}},
		{"invalid owner", []Route{{Prefix: "/", Owner: "rust"}}},
		{"missing fallback prefix", []Route{{Prefix: "/pw-api/", Owner: OwnerUpstream}}},
		{"exact route ends with slash", []Route{{Prefix: "/", Owner: OwnerUpstream}, {Prefix: "/pw-api/", Owner: OwnerUpstream, Exact: true}}},
		{"exact fallback", []Route{{Prefix: "/", Owner: OwnerUpstream, Exact: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.table); err == nil {
				t.Error("Validate() err = nil, want non-nil")
			}
		})
	}
}

// goHandlers supplies a stub for every route the table gives to Go, so this
// file does not have to be edited each time one changes hands.
func goHandlers() map[string]http.Handler {
	out := make(map[string]http.Handler)
	for _, r := range Table {
		if r.Owner == OwnerGo {
			out[r.Prefix] = stub("go " + r.Prefix)
		}
	}
	return out
}

func TestMuxRouteLongestPrefixWins(t *testing.T) {
	m, err := New(Table, stub("upstream"), goHandlers())
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"/", "/"},
		{"/scp-173", "/"},
		{"/forum:start", "/"},
		{"/-/login", "/-/"},
		{"/-/static/app.js", "/-/static/"},
		{"/pw-api/articles/scp-173/votes", "/pw-api/articles/"},
		{"/pw-api/notifications", "/pw-api/notifications"},
		{"/pw-api/notify", "/pw-api/"},
		{"/local--files/a/b.png", "/local--files/"},
		{"/local--theme/12/style.css", "/local--theme/"},
		{"/-", "/"},
		{"/pw-apidocs", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := m.Route(tt.path).Prefix; got != tt.want {
				t.Errorf("Route(%q).Prefix = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestMuxRouteExactMatchesOnlyTheWholePath(t *testing.T) {
	m, err := New(Table, stub("upstream"), goHandlers())
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"/pw-api/modules", "/pw-api/modules"},
		{"/pw-api/modules/", "/pw-api/"},
		{"/pw-api/modules/1", "/pw-api/"},
		{"/pw-api/module", "/pw-api/"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := m.Route(tt.path).Prefix; got != tt.want {
				t.Errorf("Route(%q).Prefix = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestNewRejectsNilUpstream(t *testing.T) {
	if _, err := New(Table, nil, nil); err == nil {
		t.Error("New(upstream=nil) err = nil, want non-nil")
	}
}

func TestNewRejectsGoRouteWithoutHandler(t *testing.T) {
	table := []Route{{Prefix: "/", Owner: OwnerUpstream}, {Prefix: "/pw-api/", Owner: OwnerGo}}
	if _, err := New(table, stub("upstream"), nil); err == nil {
		t.Error("New() err = nil, want non-nil")
	}
}

func TestNewRejectsUpstreamRouteWithHandler(t *testing.T) {
	table := []Route{{Prefix: "/", Owner: OwnerUpstream}, {Prefix: "/pw-api/", Owner: OwnerUpstream}}
	handlers := map[string]http.Handler{"/pw-api/": stub("go")}
	if _, err := New(table, stub("upstream"), handlers); err == nil {
		t.Error("New() err = nil, want non-nil")
	}
}

func TestNewRejectsHandlerOutsideTable(t *testing.T) {
	table := []Route{{Prefix: "/", Owner: OwnerUpstream}}
	handlers := map[string]http.Handler{"/forum/": stub("go")}
	if _, err := New(table, stub("upstream"), handlers); err == nil {
		t.Error("New() err = nil, want non-nil")
	}
}

func TestMuxServeHTTPDispatches(t *testing.T) {
	table := []Route{
		{Prefix: "/", Owner: OwnerUpstream},
		{Prefix: "/pw-api/", Owner: OwnerUpstream},
		{Prefix: "/pw-api/modules", Owner: OwnerGo, Exact: true},
	}
	m, err := New(table, stub("upstream"), map[string]http.Handler{"/pw-api/modules": stub("go")})
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"/pw-api/modules", "go"},
		{"/pw-api/articles", "upstream"},
		{"/scp-173", "upstream"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if got := rec.Body.String(); got != tt.want {
				t.Errorf("ServeHTTP(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
