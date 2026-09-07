package admin

import (
	"net/http"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type navItem struct {
	Href   string
	Label  string
	Icon   string
	Active bool
}

type navGroup struct {
	Key   string
	Label string
	Icon  string
	Items []navItem
	Open  bool
}

type layoutView struct {
	SiteTitle string
	AdminURL  string
	Here      string
	User      string
	Title     string
	Dashboard navItem
	Groups    []navGroup
	Open      *navGroup
	Body      string

	loc *i18n.Localizer
}

func (v layoutView) T(id string, args ...any) string { return v.loc.T(id, args...) }

func (h *Handler) layout(r *http.Request, loc *i18n.Localizer, title, body string) layoutView {
	ctx := r.Context()
	granted := grantsFrom(ctx)
	active, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, Prefix), "/")

	v := layoutView{
		Title:    title,
		Body:     body,
		AdminURL: Prefix,
		Here:     r.URL.Path,
		loc:      loc,
	}
	if current := site.FromContext(ctx); current != nil {
		v.SiteTitle = current.Title
	}
	if user := auth.FromContext(ctx); user != nil {
		v.User = user.DisplayLabel()
	}
	v.Dashboard = navItem{
		Href:   Prefix,
		Label:  loc.T("admin.dashboard"),
		Icon:   "fa-tachometer-alt",
		Active: active == "",
	}

	labels := make(map[string]string, len(screens))
	needs := make(map[string]string, len(screens))
	for _, s := range screens {
		labels[s.slug] = s.label
		needs[s.slug] = s.need
	}
	for _, g := range groups {
		group := navGroup{Key: g.key, Label: loc.T(g.label), Icon: g.icon}
		for _, e := range g.entries {
			label, ok := labels[e.slug]
			if !ok || !granted.Has(needs[e.slug]) {
				continue
			}
			item := navItem{
				Href:   Prefix + e.slug + "/",
				Label:  loc.T(label),
				Icon:   e.icon,
				Active: e.slug == active,
			}
			if item.Active {
				group.Open = true
			}
			group.Items = append(group.Items, item)
		}
		if len(group.Items) > 0 {
			v.Groups = append(v.Groups, group)
		}
	}

	if q := r.URL.Query(); q.Has("g") {
		want := q.Get("g")
		for i := range v.Groups {
			v.Groups[i].Open = v.Groups[i].Key == want
		}
	}
	for i := range v.Groups {
		if v.Groups[i].Open {
			v.Open = &v.Groups[i]
			break
		}
	}
	return v
}
