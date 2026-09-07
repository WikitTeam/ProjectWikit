package admin

import (
	"net/http"
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const suspiciousSlug = "suspicious"

func init() {
	register(screen{slug: suspiciousSlug, label: "admin.suspicious", need: perms.ViewSensitiveInfo, serve: (*Handler).suspicious})
}

func (h *Handler) suspicious(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if r.Method == http.MethodPost {
		return h.clearAddresses(w, r, loc)
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	return h.suspiciousScreen(w, r, loc, 0)
}

func (h *Handler) suspiciousScreen(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, cleared int64) error {
	total, err := h.deps.DB.AddressCount(r.Context())
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.suspicious"), "suspicious.html", map[string]any{
		"Total":   total,
		"Cleared": cleared,
		"CSRF":    csrf.Issue(w, r),
		"Action":  Prefix + suspiciousSlug + "/",
	})
}

func (h *Handler) clearAddresses(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if !h.verified(w, r) {
		return nil
	}
	if r.PostFormValue("confirm") == "" {
		return h.suspiciousScreen(w, r, loc, 0)
	}
	var only *int64
	if raw := r.PostFormValue("user"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return h.suspiciousScreen(w, r, loc, 0)
		}
		only = &id
	}
	gone, err := h.deps.DB.ClearAddresses(r.Context(), only)
	if err != nil {
		return err
	}
	h.note(r, db.AdminDeleted, suspiciousSlug, "", strconv.FormatInt(gone, 10))
	return h.suspiciousScreen(w, r, loc, gone)
}
