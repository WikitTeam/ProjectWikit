package admin

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const themeSlug = "themes"

var slugPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func init() {
	register(screen{slug: themeSlug, label: "admin.themes", need: perms.ManageSite, serve: (*Handler).themes})
}

func (h *Handler) themes(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+themeSlug), "/")
	if r.Method == http.MethodPost {
		return h.saveThemeForm(w, r, loc, rest)
	}
	if rest == "" {
		return h.themeList(w, r, loc)
	}
	return h.themeForm(w, r, loc, rest, "")
}

func (h *Handler) themeList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	found, err := h.deps.DB.Themes(r.Context())
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.themes"), "theme_list.html", map[string]any{
		"Themes": found,
		"New":    Prefix + themeSlug + "/new",
		"Base":   Prefix + themeSlug + "/",
	})
}

func (h *Handler) themeForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, message string) error {
	row := db.ThemeRow{Mode: db.ThemeInline}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.Theme(r.Context(), id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	return h.page(w, r, loc, loc.T("admin.themes"), "theme_form.html", map[string]any{
		"Theme":  row,
		"CSRF":   csrf.Issue(w, r),
		"Error":  message,
		"Action": Prefix + themeSlug + "/" + rest,
		"Back":   Prefix + themeSlug + "/",
		"Modes":  []string{db.ThemeInline, db.ThemeExternal},
	})
}

func (h *Handler) saveThemeForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	current := site.FromContext(r.Context())
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil
	}

	row := db.ThemeRow{
		Name:        strings.TrimSpace(r.PostFormValue("name")),
		Slug:        strings.TrimSpace(r.PostFormValue("slug")),
		Mode:        r.PostFormValue("mode"),
		CSS:         r.PostFormValue("css"),
		ExternalURL: strings.TrimSpace(r.PostFormValue("external_url")),
	}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row.ID = id
	}
	if r.PostFormValue("delete") != "" && row.ID != 0 {
		if err := h.deps.DB.DeleteTheme(r.Context(), row.ID); err != nil {
			return err
		}
		h.noteID(r, db.AdminDeleted, themeSlug, row.ID, row.Name)
		redirect(w, Prefix+themeSlug+"/")
		return nil
	}

	if problem := h.checkTheme(loc, row); problem != "" {
		return h.themeForm(w, r, loc, rest, problem)
	}
	did := db.AdminChanged
	if row.ID == 0 {
		did = db.AdminCreated
	}
	id, err := h.deps.DB.SaveTheme(r.Context(), row)
	if err != nil {
		return err
	}
	row.ID = id
	if err := h.writeThemeCSS(row); err != nil {
		return err
	}
	h.noteID(r, did, themeSlug, row.ID, row.Name)
	redirect(w, Prefix+themeSlug+"/")
	return nil
}

func (h *Handler) checkTheme(loc *i18n.Localizer, row db.ThemeRow) string {
	switch {
	case row.Name == "":
		return loc.T("admin.theme-no-name")
	case !slugPattern.MatchString(row.Slug):
		return loc.T("admin.theme-bad-slug")
	case row.Mode != db.ThemeInline && row.Mode != db.ThemeExternal:
		return loc.T("admin.theme-bad-mode")
	case row.Mode == db.ThemeExternal && row.ExternalURL == "":
		return loc.T("admin.theme-no-url")
	}
	return ""
}

func (h *Handler) writeThemeCSS(row db.ThemeRow) error {
	if row.Mode != db.ThemeInline || row.Slug == "" {
		return nil
	}
	dir := filepath.Join(h.deps.Files, "theme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(dir, row.Slug+".css")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, []byte(row.CSS), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func redirect(w http.ResponseWriter, to string) {
	seeOther(w, to, http.StatusSeeOther)
}

func seeOther(w http.ResponseWriter, to string, status int) {
	w.Header().Set("Location", to)
	w.WriteHeader(status)
}
