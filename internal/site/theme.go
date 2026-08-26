package site

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

const DefaultThemeURL = static.Prefix + "theme.css"

// ThemeURL is the stylesheet a page loads after the base one. A theme that is
// gone or points at nothing falls back rather than leaving the page unstyled.
func ThemeURL(t *db.Theme) string {
	if t == nil {
		return DefaultThemeURL
	}
	if t.Mode == db.ThemeExternal {
		if url := strings.TrimSpace(t.ExternalURL); url != "" {
			return url
		}
		return DefaultThemeURL
	}
	return "/-/theme/" + t.Slug + ".css?v=" + strconv.FormatInt(t.UpdatedAt.Unix(), 10)
}
