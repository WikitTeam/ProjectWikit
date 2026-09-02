package site

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

// A site with no usable theme loads nothing, so the base stylesheet stays the
// only thing under the site's own CSS, which is what a wikidot theme expects.
func ThemeURL(t *db.Theme) string {
	if t == nil {
		return ""
	}
	if t.Mode == db.ThemeExternal {
		return strings.TrimSpace(t.ExternalURL)
	}
	return "/-/theme/" + t.Slug + ".css?v=" + strconv.FormatInt(t.UpdatedAt.Unix(), 10)
}

const (
	ThemePrefix = "/-/theme/"
	themeSuffix = ".css"

	themeMime     = "text/css; charset=utf-8"
	themeCache    = "public, max-age=31536000, immutable"
	themeNotFound = "theme not found"
)

type ThemeFiles struct{ dir string }

var _ http.Handler = (*ThemeFiles)(nil)

// NewThemeFiles serves out of the theme directory under root, which is files/.
func NewThemeFiles(root string) *ThemeFiles {
	return &ThemeFiles{dir: filepath.Join(root, "theme")}
}

func (h *ThemeFiles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	name, ok := strings.CutPrefix(r.URL.Path, ThemePrefix)
	if !ok {
		themeMissing(w)
		return
	}
	slug := themeSlug(strings.TrimSuffix(name, themeSuffix))
	if slug == "" || !strings.HasSuffix(name, themeSuffix) {
		themeMissing(w)
		return
	}

	full, err := paths.Resolve(h.dir, slug+themeSuffix)
	if err != nil {
		themeMissing(w)
		return
	}
	body, err := os.ReadFile(full)
	if err != nil {
		themeMissing(w)
		return
	}

	w.Header().Set("Content-Type", themeMime)
	w.Header().Set("Cache-Control", themeCache)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

// Everything outside the set is dropped rather than refused, so a slug that
// picks up a stray character still finds its file.
func themeSlug(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func themeMissing(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(themeNotFound)))
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, themeNotFound)
}
