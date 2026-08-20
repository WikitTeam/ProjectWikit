package routing

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

type Owner string

const (
	OwnerGo     Owner = "go"
	OwnerDjango Owner = "django"
)

type Route struct {
	Prefix string
	Owner  Owner
	Label  string
}

const root = "/"

var Table = []Route{
	{Prefix: "/", Owner: OwnerDjango, Label: "文章"},
	{Prefix: "/-/", Owner: OwnerDjango, Label: "系统页"},
	{Prefix: "/-/static/", Owner: OwnerDjango, Label: "静态资源"},
	{Prefix: "/api/", Owner: OwnerDjango, Label: "API"},
	{Prefix: "/local--files/", Owner: OwnerDjango, Label: "站点文件"},
	{Prefix: "/local--code/", Owner: OwnerDjango, Label: "代码块"},
	{Prefix: "/local--html/", Owner: OwnerDjango, Label: "内嵌 HTML"},
	{Prefix: "/local--theme/", Owner: OwnerDjango, Label: "页面主题"},
}

func Validate(table []Route) error {
	seen := make(map[string]bool, len(table))
	hasRoot := false
	for _, r := range table {
		if !strings.HasPrefix(r.Prefix, "/") {
			return fmt.Errorf("路由前缀 %q 不以 / 开头", r.Prefix)
		}
		if !strings.HasSuffix(r.Prefix, "/") {
			return fmt.Errorf("路由前缀 %q 不以 / 结尾", r.Prefix)
		}
		if seen[r.Prefix] {
			return fmt.Errorf("路由前缀 %q 重复", r.Prefix)
		}
		seen[r.Prefix] = true
		if r.Owner != OwnerGo && r.Owner != OwnerDjango {
			return fmt.Errorf("路由 %q 的 owner = %q，期望 go 或 django", r.Prefix, r.Owner)
		}
		if r.Prefix == root {
			hasRoot = true
		}
	}
	if !hasRoot {
		return fmt.Errorf("路由表缺少兜底前缀 %q", root)
	}
	return nil
}

type Mux struct {
	routes   []Route
	fallback Route
	handlers map[string]http.Handler
}

var _ http.Handler = (*Mux)(nil)

func New(table []Route, django http.Handler, goHandlers map[string]http.Handler) (*Mux, error) {
	if err := Validate(table); err != nil {
		return nil, err
	}
	if django == nil {
		return nil, fmt.Errorf("django 转发处理器为 nil")
	}

	handlers := make(map[string]http.Handler, len(table))
	var fallback Route
	for _, r := range table {
		switch r.Owner {
		case OwnerDjango:
			if _, ok := goHandlers[r.Prefix]; ok {
				return nil, fmt.Errorf("路由 %q 的 owner = django，却注册了 Go 处理器", r.Prefix)
			}
			handlers[r.Prefix] = django
		case OwnerGo:
			h, ok := goHandlers[r.Prefix]
			if !ok {
				return nil, fmt.Errorf("路由 %q 的 owner = go，却没有注册处理器", r.Prefix)
			}
			handlers[r.Prefix] = h
		}
		if r.Prefix == root {
			fallback = r
		}
	}
	for prefix := range goHandlers {
		if _, ok := handlers[prefix]; !ok {
			return nil, fmt.Errorf("注册了 Go 处理器的前缀 %q 不在路由表里", prefix)
		}
	}

	routes := slices.Clone(table)
	slices.SortFunc(routes, func(a, b Route) int { return len(b.Prefix) - len(a.Prefix) })
	return &Mux{routes: routes, fallback: fallback, handlers: handlers}, nil
}

func (m *Mux) Route(path string) Route {
	for _, r := range m.routes {
		if strings.HasPrefix(path, r.Prefix) {
			return r
		}
	}
	return m.fallback
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.handlers[m.Route(r.URL.Path).Prefix].ServeHTTP(w, r)
}
