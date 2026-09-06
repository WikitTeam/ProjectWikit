package admin

import (
	"net/http"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const actionLogSlug = "action-log"

func init() {
	register(screen{slug: actionLogSlug, label: "admin.action-log", need: perms.ViewActionsLog, serve: (*Handler).actionLog})
}

func (h *Handler) actionLog(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	ctx := r.Context()
	kinds, err := h.deps.DB.ActionLogTypes(ctx)
	if err != nil {
		return err
	}
	kind := r.URL.Query().Get("type")
	if !contains(kinds, kind) {
		kind = ""
	}
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	found, total, err := h.deps.DB.AdminActionLog(ctx, kind, perPage, (page-1)*perPage)
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.action-log"), "action_log.html", map[string]any{
		"Entries": found,
		"Kinds":   kinds,
		"Kind":    kind,
		"Page":    page,
		"Pages":   (total + perPage - 1) / perPage,
		"Total":   total,
		"SeeIP":   granted.Has(perms.ViewSensitiveInfo),
		"Action":  Prefix + actionLogSlug + "/",
	})
}
