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
		t.Errorf("Validate(Table) err = %v，期望 nil", err)
	}
}

func TestValidateRejectsBadTable(t *testing.T) {
	tests := []struct {
		name  string
		table []Route
	}{
		{"前缀不以 / 开头", []Route{{Prefix: "/", Owner: OwnerDjango}, {Prefix: "api/", Owner: OwnerDjango}}},
		{"前缀不以 / 结尾", []Route{{Prefix: "/", Owner: OwnerDjango}, {Prefix: "/api", Owner: OwnerDjango}}},
		{"前缀重复", []Route{{Prefix: "/", Owner: OwnerDjango}, {Prefix: "/api/", Owner: OwnerDjango}, {Prefix: "/api/", Owner: OwnerGo}}},
		{"owner 非法", []Route{{Prefix: "/", Owner: "rust"}}},
		{"缺兜底前缀", []Route{{Prefix: "/api/", Owner: OwnerDjango}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.table); err == nil {
				t.Error("Validate() err = nil，期望非 nil")
			}
		})
	}
}

func TestMuxRouteLongestPrefixWins(t *testing.T) {
	m, err := New(Table, stub("django"), nil)
	if err != nil {
		t.Fatalf("New() err = %v，期望 nil", err)
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
		{"/api/articles/scp-173/votes", "/api/"},
		{"/local--files/a/b.png", "/local--files/"},
		{"/local--theme/12/style.css", "/local--theme/"},
		{"/-", "/"},
		{"/apidocs", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := m.Route(tt.path).Prefix; got != tt.want {
				t.Errorf("Route(%q).Prefix = %q，期望 %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestNewRejectsNilDjango(t *testing.T) {
	if _, err := New(Table, nil, nil); err == nil {
		t.Error("New(django=nil) err = nil，期望非 nil")
	}
}

func TestNewRejectsGoRouteWithoutHandler(t *testing.T) {
	table := []Route{{Prefix: "/", Owner: OwnerDjango}, {Prefix: "/api/", Owner: OwnerGo}}
	if _, err := New(table, stub("django"), nil); err == nil {
		t.Error("New() err = nil，期望非 nil")
	}
}

func TestNewRejectsDjangoRouteWithHandler(t *testing.T) {
	table := []Route{{Prefix: "/", Owner: OwnerDjango}, {Prefix: "/api/", Owner: OwnerDjango}}
	handlers := map[string]http.Handler{"/api/": stub("go")}
	if _, err := New(table, stub("django"), handlers); err == nil {
		t.Error("New() err = nil，期望非 nil")
	}
}

func TestNewRejectsHandlerOutsideTable(t *testing.T) {
	table := []Route{{Prefix: "/", Owner: OwnerDjango}}
	handlers := map[string]http.Handler{"/forum/": stub("go")}
	if _, err := New(table, stub("django"), handlers); err == nil {
		t.Error("New() err = nil，期望非 nil")
	}
}

func TestMuxServeHTTPDispatches(t *testing.T) {
	table := []Route{
		{Prefix: "/", Owner: OwnerDjango},
		{Prefix: "/api/", Owner: OwnerGo},
	}
	m, err := New(table, stub("django"), map[string]http.Handler{"/api/": stub("go")})
	if err != nil {
		t.Fatalf("New() err = %v，期望 nil", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"/api/articles", "go"},
		{"/scp-173", "django"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if got := rec.Body.String(); got != tt.want {
				t.Errorf("ServeHTTP(%q) = %q，期望 %q", tt.path, got, tt.want)
			}
		})
	}
}
