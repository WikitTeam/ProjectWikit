package routing

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

type Owner string

const (
	OwnerGo       Owner = "go"
	OwnerUpstream Owner = "upstream"
)

type Route struct {
	Prefix string
	Owner  Owner
	Label  string

	Exact bool
}

func (r Route) matches(path string) bool {
	if r.Exact {
		return path == r.Prefix
	}
	return strings.HasPrefix(path, r.Prefix)
}

const root = "/"

var Table = []Route{
	{Prefix: "/", Owner: OwnerGo, Label: "articles"},
	{Prefix: "/-/", Owner: OwnerUpstream, Label: "system pages"},
	{Prefix: "/-/static/", Owner: OwnerGo, Label: "static assets"},
	{Prefix: "/-/theme/", Owner: OwnerGo, Label: "site theme"},
	{Prefix: "/-/users/", Owner: OwnerGo, Label: "user profiles"},
	{Prefix: "/-/login", Owner: OwnerGo, Label: "sign in", Exact: true},
	{Prefix: "/-/logout", Owner: OwnerGo, Label: "sign out", Exact: true},
	{Prefix: "/-/signup", Owner: OwnerGo, Label: "sign up", Exact: true},
	{Prefix: "/-/signup/", Owner: OwnerGo, Label: "sign up"},
	{Prefix: "/-/accept/", Owner: OwnerGo, Label: "invitation"},
	{Prefix: "/-/password_reset", Owner: OwnerGo, Label: "password reset", Exact: true},
	{Prefix: "/-/password_reset/", Owner: OwnerGo, Label: "password reset"},
	{Prefix: "/-/reset/", Owner: OwnerGo, Label: "password reset"},
	{Prefix: "/-/tickets/submit", Owner: OwnerGo, Label: "ticket form", Exact: true},
	{Prefix: "/-/membership/password", Owner: OwnerGo, Label: "membership form", Exact: true},
	{Prefix: "/-/profile/edit", Owner: OwnerGo, Label: "profile form", Exact: true},
	{Prefix: "/-/favourites", Owner: OwnerGo, Label: "saved pages", Exact: true},
	{Prefix: "/-/ratings", Owner: OwnerGo, Label: "own ratings", Exact: true},
	{Prefix: "/-/liked-posts", Owner: OwnerGo, Label: "own likes", Exact: true},
	{Prefix: "/-/notifications", Owner: OwnerGo, Label: "notification page", Exact: true},
	{Prefix: "/-/notifications/", Owner: OwnerGo, Label: "notification page"},
	{Prefix: "/-/messages", Owner: OwnerGo, Label: "message page", Exact: true},
	{Prefix: "/-/messages/", Owner: OwnerGo, Label: "message page"},
	{Prefix: "/pw-api/", Owner: OwnerUpstream, Label: "API"},
	{Prefix: "/pw-api/modules", Owner: OwnerGo, Label: "module API", Exact: true},
	{Prefix: "/pw-api/preview", Owner: OwnerGo, Label: "editor preview", Exact: true},
	{Prefix: "/pw-api/articles", Owner: OwnerGo, Label: "article listing", Exact: true},
	{Prefix: "/pw-api/articles/", Owner: OwnerGo, Label: "article API"},
	{Prefix: "/pw-api/files/", Owner: OwnerGo, Label: "file API"},
	{Prefix: "/pw-api/notifications", Owner: OwnerGo, Label: "notification API", Exact: true},
	{Prefix: "/pw-api/notifications/subscribe", Owner: OwnerGo, Label: "subscription API", Exact: true},
	{Prefix: "/pw-api/messages/", Owner: OwnerGo, Label: "message API"},
	{Prefix: "/pw-api/users", Owner: OwnerGo, Label: "user API", Exact: true},
	{Prefix: "/pw-api/users/", Owner: OwnerGo, Label: "user API"},
	{Prefix: "/pw-api/admin/", Owner: OwnerGo, Label: "admin API"},
	{Prefix: "/pw-api/favourites", Owner: OwnerGo, Label: "saved page API", Exact: true},
	{Prefix: "/pw-api/ratings", Owner: OwnerGo, Label: "own rating API", Exact: true},
	{Prefix: "/pw-api/liked-posts", Owner: OwnerGo, Label: "own like API", Exact: true},
	{Prefix: "/local--files/", Owner: OwnerGo, Label: "site files"},
	{Prefix: "/local--resized-images/", Owner: OwnerGo, Label: "scaled images"},
	{Prefix: "/local--code/", Owner: OwnerGo, Label: "code blocks"},
	{Prefix: "/local--html/", Owner: OwnerGo, Label: "embedded HTML"},
	{Prefix: "/local--theme/", Owner: OwnerGo, Label: "page theme"},
}

func Validate(table []Route) error {
	seen := make(map[string]bool, len(table))
	hasRoot := false
	for _, r := range table {
		if !strings.HasPrefix(r.Prefix, "/") {
			return fmt.Errorf("route prefix %q does not start with /", r.Prefix)
		}
		if r.Exact && strings.HasSuffix(r.Prefix, "/") {
			return fmt.Errorf("exact route %q ends with /", r.Prefix)
		}
		if !r.Exact && !strings.HasSuffix(r.Prefix, "/") {
			return fmt.Errorf("route prefix %q does not end with /", r.Prefix)
		}
		if seen[r.Prefix] {
			return fmt.Errorf("duplicate route prefix %q", r.Prefix)
		}
		seen[r.Prefix] = true
		if r.Owner != OwnerGo && r.Owner != OwnerUpstream {
			return fmt.Errorf("route %q owner = %q, want go or upstream", r.Prefix, r.Owner)
		}
		if r.Prefix == root {
			if r.Exact {
				return fmt.Errorf("route %q is exact, which leaves the table without a fallback", r.Prefix)
			}
			hasRoot = true
		}
	}
	if !hasRoot {
		return fmt.Errorf("route table has no fallback prefix %q", root)
	}
	return nil
}

type Mux struct {
	routes   []Route
	fallback Route
	handlers map[string]http.Handler
}

var _ http.Handler = (*Mux)(nil)

func New(table []Route, upstream http.Handler, goHandlers map[string]http.Handler) (*Mux, error) {
	if err := Validate(table); err != nil {
		return nil, err
	}
	if upstream == nil {
		return nil, fmt.Errorf("upstream handler is nil")
	}

	handlers := make(map[string]http.Handler, len(table))
	var fallback Route
	for _, r := range table {
		switch r.Owner {
		case OwnerUpstream:
			if _, ok := goHandlers[r.Prefix]; ok {
				return nil, fmt.Errorf("route %q is owned by upstream but a Go handler is registered", r.Prefix)
			}
			handlers[r.Prefix] = upstream
		case OwnerGo:
			h, ok := goHandlers[r.Prefix]
			if !ok {
				return nil, fmt.Errorf("route %q is owned by go but no handler is registered", r.Prefix)
			}
			handlers[r.Prefix] = h
		}
		if r.Prefix == root {
			fallback = r
		}
	}
	for prefix := range goHandlers {
		if _, ok := handlers[prefix]; !ok {
			return nil, fmt.Errorf("prefix %q has a Go handler but is not in the route table", prefix)
		}
	}

	routes := slices.Clone(table)
	slices.SortFunc(routes, func(a, b Route) int { return len(b.Prefix) - len(a.Prefix) })
	return &Mux{routes: routes, fallback: fallback, handlers: handlers}, nil
}

func (m *Mux) Route(path string) Route {
	for _, r := range m.routes {
		if r.matches(path) {
			return r
		}
	}
	return m.fallback
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.handlers[m.Route(r.URL.Path).Prefix].ServeHTTP(w, r)
}
