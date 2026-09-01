package site

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
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
