package admin

import (
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type navItem struct {
	Href   string
	Label  string
	Icon   string
	Active bool
	Search string
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
	Updates   []updateNotice

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

	byslug := make(map[string]screen, len(screens))
	for _, s := range screens {
		byslug[s.slug] = s
	}
	for _, g := range groups {
		group := navGroup{Key: g.key, Label: loc.T(g.label), Icon: g.icon}
		for _, e := range g.entries {
			s, ok := byslug[e.slug]
			if !ok || !s.opensFor(granted, instanceSuper(ctx)) {
				continue
			}
			item := navItem{
				Href:   Prefix + e.slug + "/",
				Label:  loc.T(s.label),
				Icon:   e.icon,
				Active: e.slug == active,
				Search: searchText(loc, e.slug),
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

// screenTemplates names what each screen renders, so the menu search finds a
// screen by the fields and sections on it and not only by its title.
var screenTemplates = map[string][]string{
	accessSlug:         {"access.html"},
	adminLogSlug:       {"admin_log.html"},
	categorySlug:       {"forum_category_form.html", "forum_category_list.html"},
	inviteSlug:         {"invite_list.html"},
	announcementSlug:   {"announcement.html", "announcement_detail.html"},
	ipBanSlug:          {"ip_ban_list.html"},
	membershipSlug:     {"ticket_list.html", "ticket_form.html"},
	membershipQuizSlug: {"membership_quiz.html"},
	pluginSlug:         {"plugins.html"},
	mirrorSlug:         {"mirror_form.html", "mirror_list.html"},
	pageCategorySlug:   {"page_category_form.html", "page_category_list.html"},
	pageSlug:           {"page_list.html", "page_edit.html"},
	recentPostSlug:     {"forum_posts.html"},
	reportSlug:         {"report_list.html", "report_form.html"},
	roleCategorySlug:   {"role_category.html"},
	roleSlug:           {"role_list.html", "role_form.html"},
	sectionSlug:        {"forum_section_form.html", "forum_section_list.html"},
	siteSlug:           {"site_form.html"},
	suspiciousSlug:     {"suspicious.html"},
	tagCategorySlug:    {"tag_category_form.html", "tag_category_list.html"},
	tagSlug:            {"tag_form.html", "tag_list.html"},
	themeSlug:          {"theme_form.html", "theme_list.html"},
	ticketSlug:         {"ticket_list.html", "ticket_form.html"},
	userSlug: {"user_list.html", "user_form.html", "user_new.html", "user_invite.html", "user_mail.html",
		"user_claim.html", "user_bot.html", "user_activity.html", "user_reset_votes.html"},
}

var (
	searchKeyPattern = regexp.MustCompile(`(?:"[kl]" |\bt )"(admin\.[a-z0-9-]+)"`)
	searchKeysOnce   sync.Once
	searchKeys       map[string][]string
)

// Hints read as sentences and buttons sit on every screen, so neither helps
// tell one screen from another.
func screenSearchKeys() map[string][]string {
	searchKeysOnce.Do(func() {
		searchKeys = map[string][]string{}
		seenOn := map[string]int{}
		for slug, names := range screenTemplates {
			seen := map[string]bool{}
			for _, name := range names {
				body, err := fs.ReadFile(files, "templates/"+name)
				if err != nil {
					continue
				}
				for _, m := range searchKeyPattern.FindAllStringSubmatch(string(body), -1) {
					key := m[1]
					if seen[key] || strings.HasPrefix(key, "admin.h-") || strings.HasPrefix(key, "admin.ph-") {
						continue
					}
					seen[key] = true
					searchKeys[slug] = append(searchKeys[slug], key)
					seenOn[key]++
				}
			}
		}
		for slug, keys := range searchKeys {
			kept := keys[:0]
			for _, key := range keys {
				if seenOn[key] < 4 {
					kept = append(kept, key)
				}
			}
			searchKeys[slug] = kept
		}
	})
	return searchKeys
}

func searchText(loc *i18n.Localizer, slug string) string {
	keys := screenSearchKeys()[slug]
	words := make([]string, len(keys))
	for i, key := range keys {
		words[i] = loc.T(key)
	}
	return strings.Join(words, " ")
}
