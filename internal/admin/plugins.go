package admin

import (
	"net/http"
	"slices"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const pluginSlug = "plugins"

func init() {
	register(screen{slug: pluginSlug, label: "admin.plugins", need: perms.ManageSite, serve: (*Handler).plugins})
}

type pluginRow struct {
	Name        string
	Title       string
	Version     string
	Description string
	Modules     []string
	Events      []string
	Permissions []string
	Hosts       []string
	Enabled     bool
}

func (h *Handler) plugins(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	installed := h.deps.Plugins.Installed()
	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		picked := r.PostForm["enabled"]
		for _, p := range installed {
			if err := h.deps.Plugins.SetEnabled(ctx, siteID(ctx), p.Name, slices.Contains(picked, p.Name)); err != nil {
				return err
			}
		}
		h.note(r, db.AdminChanged, pluginSlug, "", "")
		redirect(w, Prefix+pluginSlug+"/")
		return nil
	}
	enabled, err := h.deps.Plugins.Enabled(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	rows := make([]pluginRow, 0, len(installed))
	for _, p := range installed {
		row := pluginRow{Name: p.Name, Title: p.Title, Version: p.Version, Description: p.Description,
			Events: p.Events, Hosts: p.AllowedHosts, Enabled: slices.Contains(enabled, p.Name)}
		if row.Title == "" {
			row.Title = p.Name
		}
		for _, m := range p.Modules {
			row.Modules = append(row.Modules, m.Name)
		}
		for _, perm := range p.Permissions {
			row.Permissions = append(row.Permissions, loc.T("admin.plugin-perm-"+perm))
		}
		rows = append(rows, row)
	}
	return h.page(w, r, loc, loc.T("admin.plugins"), "plugins.html", map[string]any{
		"Plugins": rows,
		"CSRF":    csrf.Issue(w, r),
		"Action":  Prefix + pluginSlug + "/",
	})
}
