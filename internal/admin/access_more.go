package admin

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
)

type roleRow struct {
	Name     string
	Label    string
	Base     string
	Override string
	Result   string
	Tone     string
}

type roleGroup struct {
	Label string
	Rows  []roleRow
}

func (h *Handler) accessRole(r *http.Request, x *explainer, data map[string]any) error {
	ctx := r.Context()
	q := r.URL.Query()
	form := accessForm{Category: q.Get("category")}
	form.Role, _ = strconv.ParseInt(q.Get("role"), 10, 64)
	data["Form"] = form
	var categories []string
	for name := range x.cats {
		categories = append(categories, name)
	}
	slices.Sort(categories)
	data["Categories"] = categories
	choice, ok := x.roles[form.Role]
	if !ok {
		return nil
	}
	loaded, err := h.deps.DB.RolePermissions(ctx, []int64{form.Role})
	if err != nil {
		return err
	}
	var object *perms.Object
	if _, known := x.cats[form.Category]; known {
		overrides, err := h.deps.DB.CategoryOverrides(ctx, siteID(ctx), form.Category)
		if err != nil {
			return err
		}
		object = &perms.Object{Overrides: overrides}
		x.target = &accessTarget{category: form.Category}
	}
	subject := perms.Subject{Active: true, Roles: loaded}
	var groups []roleGroup
	for _, g := range perms.Catalog {
		group := roleGroup{Label: x.loc.T("admin.perm-group-" + g.Key)}
		for _, name := range g.Names {
			rv := perms.Explain(subject, object, name).Roles[0]
			row := roleRow{Name: name, Label: x.loc.T("admin.perm-" + name), Base: x.value(rv.Base), Override: "—", Result: x.value(rv.Result), Tone: toneOf(rv.Result)}
			if rv.HasOverride {
				row.Override = x.value(rv.Override)
			}
			group.Rows = append(group.Rows, row)
		}
		groups = append(groups, group)
	}
	data["RoleTable"] = groups
	data["RoleName"] = choice.Label()
	data["RoleHref"] = x.roleHref(form.Role)
	data["Everyone"] = slices.Contains(builtinRoles, choice.Slug)
	return nil
}

func toneOf(v perms.Value) string {
	switch v {
	case perms.Allow:
		return "allow"
	case perms.Deny:
		return "deny"
	}
	return ""
}

type holderRow struct {
	Name    string
	Href    string
	Check   string
	Allowed bool
	Summary string
}

func (h *Handler) accessHolders(r *http.Request, x *explainer, data map[string]any) error {
	ctx := r.Context()
	q := r.URL.Query()
	form := accessForm{Perm: q.Get("perm"), Target: strings.TrimSpace(q.Get("target"))}
	data["Form"] = form
	if form.Perm == "" {
		return nil
	}
	if perms.GroupOf(form.Perm) == "" {
		data["Problem"] = x.loc.T("admin.access-bad-perm")
		return nil
	}
	target, ok, err := h.findTarget(ctx, form.Target)
	if err != nil {
		return err
	}
	if !ok {
		data["Problem"] = x.loc.T("admin.access-no-target", "target", form.Target)
		return nil
	}
	x.target = target
	now := time.Now()
	p := repo.NewPerms(ctx, h.deps.DB)
	object, err := target.object(p, nil)
	if err != nil {
		return err
	}

	anonymous, err := p.Subject(nil, now)
	if err != nil {
		return err
	}
	ids, err := h.deps.DB.RoleIDsBySlug(ctx, siteID(ctx), builtinRoles)
	if err != nil {
		return err
	}
	var plainRoles []int64
	for _, slug := range builtinRoles {
		if id, ok := ids[slug]; ok {
			plainRoles = append(plainRoles, id)
		}
	}
	roles, err := h.deps.DB.RolePermissions(ctx, plainRoles)
	if err != nil {
		return err
	}
	member := perms.Subject{Active: true, ForumActive: true, Roles: roles}
	data["Anonymous"] = x.verdict(perms.Explain(anonymous, object, form.Perm), false)
	data["Member"] = x.verdict(perms.Explain(member, object, form.Perm), false)

	candidates, err := h.deps.DB.AccessCandidates(ctx, siteID(ctx), now, accessLimit+1)
	if err != nil {
		return err
	}
	if len(candidates) > accessLimit {
		candidates = candidates[:accessLimit]
		data["Truncated"] = accessLimit
	}
	holders, refused := []holderRow{}, []holderRow{}
	for _, id := range candidates {
		u, err := h.deps.DB.UserByID(ctx, id)
		if err != nil {
			if ignoreMissing(err) == nil {
				continue
			}
			return err
		}
		subject, err := p.Subject(u, now)
		if err != nil {
			return err
		}
		object, err := target.object(p, u)
		if err != nil {
			return err
		}
		if err := h.prepareFor(ctx, x, target, u, subject, now); err != nil {
			return err
		}
		v := x.verdict(perms.Explain(subject, object, form.Perm), false)
		check := Prefix + accessSlug + "/?user=" + url.QueryEscape(u.Username) + "&perm=" + url.QueryEscape(form.Perm)
		if form.Target != "" {
			check += "&target=" + url.QueryEscape(form.Target)
		}
		row := holderRow{Name: u.DisplayLabel(), Href: Prefix + userSlug + "/" + strconv.FormatInt(u.ID, 10), Check: check, Allowed: v.Allowed, Summary: v.Summary}
		if v.Allowed {
			holders = append(holders, row)
		} else {
			refused = append(refused, row)
		}
	}
	data["Holders"] = holders
	data["Refused"] = refused
	data["Checked"] = true
	return nil
}

type finding struct {
	RoleID int64
	Tone   string
	Text   string
	Href   string
	Link   string
}

func (h *Handler) accessLint(r *http.Request, x *explainer, data map[string]any) error {
	findings, err := h.lint(r.Context(), x)
	if err != nil {
		return err
	}
	data["Findings"] = findings
	data["Linted"] = true
	return nil
}

func (h *Handler) lint(ctx context.Context, x *explainer) ([]finding, error) {
	t := x.loc.T
	roleRows, err := h.deps.DB.AdminRoles(ctx, siteID(ctx))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(roleRows))
	for i, row := range roleRows {
		ids[i] = row.ID
	}
	loaded, err := h.deps.DB.RolePermissions(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := map[int64]perms.Role{}
	for _, role := range loaded {
		byID[role.ID] = role
	}
	names := func(list []string) string {
		var out []string
		for _, name := range list {
			out = append(out, x.perm(name))
		}
		return strings.Join(out, t("admin.access-list-join"))
	}
	var findings []finding
	roleLink := func(id int64) (string, string) { return x.roleHref(id), t("admin.fix-open-role") }

	for _, row := range roleRows {
		role := byID[row.ID]
		name := x.roleName(row.ID)
		shared := slices.Contains(builtinRoles, row.Slug)
		denied := live(role.Restrictions)
		href, link := roleLink(row.ID)
		if shared && len(denied) > 0 {
			findings = append(findings, finding{RoleID: row.ID, Tone: "deny", Text: t("admin.lint-shared-deny", "role", name, "perms", names(denied)), Href: href, Link: link})
		}
		if shared {
			var powerful []string
			for _, p := range live(role.Permissions) {
				if group := perms.GroupOf(p); group == "admin" || group == "members" || group == "tickets" {
					powerful = append(powerful, p)
				}
			}
			if len(powerful) > 0 {
				findings = append(findings, finding{RoleID: row.ID, Tone: "deny", Text: t("admin.lint-shared-admin", "role", name, "perms", names(powerful)), Href: href, Link: link})
			}
		}
		if !shared && len(denied) > 0 && row.Users > 0 {
			findings = append(findings, finding{RoleID: row.ID, Tone: "warn", Text: t("admin.lint-role-deny", "role", name, "perms", names(denied), "n", row.Users), Href: href, Link: link})
		}
		for _, p := range append(append([]string{}, role.Permissions...), role.Restrictions...) {
			if slices.Contains(perms.Retired, p) {
				findings = append(findings, finding{RoleID: row.ID, Tone: "info", Text: t("admin.lint-retired", "role", name, "perm", x.perm(p)), Href: href, Link: link})
			}
		}
	}

	slugOf := map[int64]string{}
	for _, row := range roleRows {
		slugOf[row.ID] = row.Slug
	}
	var categories []string
	for name := range x.cats {
		categories = append(categories, name)
	}
	slices.Sort(categories)
	for _, category := range categories {
		overrides, err := h.deps.DB.CategoryOverrides(ctx, siteID(ctx), category)
		if err != nil {
			return nil, err
		}
		x.target = &accessTarget{category: category}
		href := Prefix + pageCategorySlug + "/" + strconv.FormatInt(x.cats[category], 10)
		seen := map[int64]int{}
		for _, o := range overrides {
			seen[o.RoleID]++
			if seen[o.RoleID] > 1 {
				continue
			}
			slug := slugOf[o.RoleID]
			if denied := live(o.Restrictions); slices.Contains(builtinRoles, slug) && len(denied) > 0 {
				findings = append(findings, finding{RoleID: o.RoleID, Tone: "deny", Text: t("admin.lint-override-shared-deny", "role", x.roleName(o.RoleID), "category", x.categoryName(), "perms", names(denied)), Href: href, Link: t("admin.fix-open-category")})
			}
		}
		for id, n := range seen {
			if n > 1 {
				findings = append(findings, finding{RoleID: id, Tone: "warn", Text: t("admin.lint-override-duplicate", "role", x.roleName(id), "category", x.categoryName(), "n", n), Href: href, Link: t("admin.fix-open-category")})
			}
		}
	}
	x.target = nil

	if x.site.DefaultRoleID == nil {
		findings = append(findings, finding{Tone: "warn", Text: t("admin.lint-no-signup-role"), Href: Prefix + siteSlug + "/", Link: t("admin.site")})
	}
	if x.site.VerifiedRoleID == nil {
		findings = append(findings, finding{Tone: "warn", Text: t("admin.lint-no-claim-role"), Href: Prefix + siteSlug + "/", Link: t("admin.site")})
	}
	order := map[string]int{"deny": 0, "warn": 1, "info": 2}
	slices.SortStableFunc(findings, func(a, b finding) int { return order[a.Tone] - order[b.Tone] })
	return findings, nil
}

// A role may still list a retired name, but it no longer decides anything.
func live(list []string) []string {
	var out []string
	for _, name := range list {
		if !slices.Contains(perms.Retired, name) {
			out = append(out, name)
		}
	}
	return out
}
