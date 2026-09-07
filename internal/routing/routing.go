// Package routing sends a request to the handler that claims the longest
// matching prefix.
package routing

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

const root = "/"

type Mux struct {
	prefixes []string
	handlers map[string]http.Handler
}

var _ http.Handler = (*Mux)(nil)

// A key ending in a slash claims everything below it, one without claims that
// path alone.
func New(handlers map[string]http.Handler) (*Mux, error) {
	if handlers[root] == nil {
		return nil, fmt.Errorf("no handler for %q, which every unclaimed path falls to", root)
	}
	prefixes := make([]string, 0, len(handlers))
	for prefix, h := range handlers {
		if !strings.HasPrefix(prefix, root) {
			return nil, fmt.Errorf("route prefix %q does not start with %q", prefix, root)
		}
		if h == nil {
			return nil, fmt.Errorf("route prefix %q has no handler", prefix)
		}
		prefixes = append(prefixes, prefix)
	}
	slices.SortFunc(prefixes, func(a, b string) int { return len(b) - len(a) })

	return &Mux{prefixes: prefixes, handlers: maps.Clone(handlers)}, nil
}

func (m *Mux) Route(path string) string {
	for _, prefix := range m.prefixes {
		if strings.HasSuffix(prefix, root) {
			if strings.HasPrefix(path, prefix) {
				return prefix
			}
			continue
		}
		if path == prefix {
			return prefix
		}
	}
	return root
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.handlers[m.Route(r.URL.Path)].ServeHTTP(w, r)
}
