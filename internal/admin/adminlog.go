package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const adminLogSlug = "admin-log"

const adminLogPerPage = 60

func init() {
	register(screen{slug: adminLogSlug, label: "admin.admin-log", need: perms.ViewActionsLog, serve: (*Handler).adminLog})
}

func (h *Handler) note(r *http.Request, action, screen, target, label string) {
	entry := db.AdminNote{
		Action: action,
		Screen: screen,
		Target: target,
		Label:  label,
		SiteID: siteID(r.Context()),
		At:     time.Now().UTC(),
	}
	if by := auth.FromContext(r.Context()); by != nil {
		entry.UserID = &by.ID
		entry.Name = by.Username
	}
	if err := h.deps.DB.WriteAdminNote(r.Context(), entry); err != nil {
		h.deps.logger().Error("record admin action", "screen", screen, "action", action, "err", err)
	}
}

func (h *Handler) noteID(r *http.Request, action, screen string, id int64, label string) {
	h.note(r, action, screen, strconv.FormatInt(id, 10), label)
}

func (h *Handler) adminLog(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	ctx := r.Context()
	screen := r.URL.Query().Get("screen")
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	found, err := h.deps.DB.AdminNotes(ctx, siteID(ctx), screen, adminLogPerPage+1, (page-1)*adminLogPerPage)
	if err != nil {
		return err
	}
	more := len(found) > adminLogPerPage
	if more {
		found = found[:adminLogPerPage]
	}
	screens, err := h.deps.DB.AdminNoteScreens(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.admin-log"), "admin_log.html", map[string]any{
		"Notes":   found,
		"Screens": screens,
		"Screen":  screen,
		"Page":    page,
		"More":    more,
		"Action":  Prefix + adminLogSlug + "/",
	})
}

func screenLabel(slug string) string {
	for _, s := range screens {
		if s.slug == slug {
			return s.label
		}
	}
	return ""
}
