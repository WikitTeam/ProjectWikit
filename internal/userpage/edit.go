package userpage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/pageconfig"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const EditPrefix = "/-/profile/edit"

const loginPath = "/-/login"

const (
	maxAvatarBytes  = 5 << 20
	avatarMemory    = 1 << 20
	avatarDirectory = "-/users"
)

// The extension comes from what the bytes are rather than from what the upload
// called itself, so a name ending in .png cannot decide how the file is served.
var avatarTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type EditHandler struct {
	deps Deps
}

var _ http.Handler = (*EditHandler)(nil)

func NewEdit(d Deps) *EditHandler { return &EditHandler{deps: d} }

func (h *EditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	current := site.FromContext(ctx)
	viewer := auth.FromContext(ctx)
	if current == nil || viewer == nil {
		redirect(w, loginPath+"?to="+url.QueryEscape(EditPrefix), http.StatusFound)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)

	token := csrf.Issue(w, r)

	var problem string
	if r.Method == http.MethodPost {
		var err error
		problem, err = h.handle(w, r, loc, current, viewer)
		switch {
		case errors.Is(err, errBadToken):
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		case err != nil:
			h.deps.logger().Error("save profile", "user", viewer.ID, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case problem == "":
			redirect(w, EditPrefix+"?saved=1", http.StatusSeeOther)
			return
		}
	}

	body, err := h.page(r, loc, current, viewer, token, problem)
	if err != nil {
		h.deps.logger().Error("render profile form", "user", viewer.ID, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
}

// Not http.Redirect, which writes an HTML body for GET where this response has
// none.
func redirect(w http.ResponseWriter, location string, status int) {
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(status)
}

var errBadToken = errors.New("userpage: the form carries no valid csrf token")

func (h *EditHandler) handle(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	current *db.Site, viewer *db.User) (string, error) {

	if r.ContentLength > maxAvatarBytes {
		return loc.T("profile.error-avatar-size"), nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes)
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		return "", errBadToken
	}
	if err := r.ParseMultipartForm(avatarMemory); err != nil {
		return loc.T("profile.error-avatar-size"), nil
	}

	avatar, problem, err := h.avatar(r, loc, viewer)
	if problem != "" || err != nil {
		return problem, err
	}

	first, last := splitFullName(r.PostFormValue("full_name"))
	bio := strings.TrimSpace(r.PostFormValue("bio"))
	if err := h.deps.DB.UpdateProfile(r.Context(), viewer.ID, first, last, bio, avatar); err != nil {
		return "", err
	}
	return "", h.deps.DB.SetUserPreference(r.Context(), viewer.ID,
		pageconfig.PreferenceSection, pageconfig.PreferenceAdvancedSourceEditor,
		pageconfig.PreferenceValue(r.PostFormValue("advanced_editor") != ""))
}

func (h *EditHandler) avatar(r *http.Request, loc *i18n.Localizer, viewer *db.User) (*string, string, error) {
	file, _, err := r.FormFile("avatar")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, "", nil
	}
	if err != nil {
		return nil, loc.T("profile.error-avatar-type"), nil
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", nil
	}
	if len(data) > maxAvatarBytes {
		return nil, loc.T("profile.error-avatar-size"), nil
	}
	ext, ok := avatarTypes[mediaType(http.DetectContentType(data))]
	if !ok {
		return nil, loc.T("profile.error-avatar-type"), nil
	}

	tag := make([]byte, 8)
	if _, err := rand.Read(tag); err != nil {
		return nil, "", err
	}
	stored := avatarDirectory + "/" + strconv.FormatInt(viewer.ID, 10) + "-" + hex.EncodeToString(tag) + ext

	full := filepath.Join(h.deps.Files, filepath.FromSlash(stored))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return nil, "", err
	}
	return &stored, "", nil
}

func mediaType(header string) string {
	kind, _, _ := strings.Cut(header, ";")
	return strings.TrimSpace(kind)
}

func splitFullName(value string) (first, last string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if i := strings.LastIndex(value, " "); i >= 0 {
		return value[:i], value[i+1:]
	}
	return value, ""
}

func (h *EditHandler) page(r *http.Request, loc *i18n.Localizer, current *db.Site,
	viewer *db.User, token, problem string) (string, error) {

	profile, err := h.deps.DB.ProfileByID(r.Context(), viewer.ID)
	if err != nil {
		return "", err
	}

	raw, err := h.deps.DB.UserPreference(r.Context(), viewer.ID,
		pageconfig.PreferenceSection, pageconfig.PreferenceAdvancedSourceEditor)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}

	data := shell.ProfileEdit{
		AuthIcon:       authIcon(current),
		DisplayName:    displayName(profile),
		Avatar:         avatar(profile),
		ProfileURL:     Prefix + url.PathEscape(profile.Username),
		FullName:       strings.TrimSpace(profile.FirstName + " " + profile.LastName),
		Bio:            profile.Bio,
		AdvancedEditor: pageconfig.PreferenceEnabled(raw),
		CSRF:           token,
		Error:          problem,
		Saved:          problem == "" && r.URL.Query().Get("saved") == "1",
	}

	render := shell.New(loc, h.deps.Assets, h.deps.TimeZone)
	content, err := render.ProfileEdit(data)
	if err != nil {
		return "", err
	}
	theme, err := site.ThemeURLByID(r.Context(), h.deps.DB, current.SystemThemeID)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	err = render.SystemPage(&out, shell.System{
		Title:     loc.T("profile.edit"),
		ThemeURL:  theme,
		BodyClass: "wikit-page",
		Content:   content,
	})
	return out.String(), err
}
