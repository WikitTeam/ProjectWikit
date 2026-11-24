package storage

import (
	"net/url"
	"strings"
)

type Mode string

const (
	ModeLocal  Mode = "local"
	ModeProxy  Mode = "proxy"
	ModeDirect Mode = "direct"
)

func ParseMode(raw string) (Mode, bool) {
	switch m := Mode(strings.ToLower(strings.TrimSpace(raw))); m {
	case "", ModeLocal:
		return ModeLocal, true
	case ModeProxy, ModeDirect:
		return m, true
	}
	return "", false
}

type Backend struct {
	Name      string
	Store     Store
	Mode      Mode
	PublicURL string
}

// Only a direct backend hands readers an address outside pwikit.
func (b *Backend) URL(key string) (string, bool) {
	if b == nil || b.Mode != ModeDirect || b.PublicURL == "" {
		return "", false
	}
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.TrimSuffix(b.PublicURL, "/") + "/" + strings.Join(parts, "/"), true
}

// Files under "-/" belong to the instance rather than to one site, so they
// stay with the shared backend whichever site the request came through.
type Set struct {
	Shared *Backend
	Sites  map[string]*Backend
}

func Single(b *Backend) *Set {
	return &Set{Shared: b}
}

func (s *Set) For(slug string) *Backend {
	if b, ok := s.Sites[slug]; ok {
		return b
	}
	return s.Shared
}

func (s *Set) ForKey(slug, key string) *Backend {
	if strings.HasPrefix(key, "-/") {
		return s.Shared
	}
	return s.For(slug)
}

func (s *Set) All() []*Backend {
	out := []*Backend{s.Shared}
	for _, b := range s.Sites {
		out = append(out, b)
	}
	return out
}

func IsShared(key string) bool {
	return strings.HasPrefix(key, "-/")
}
