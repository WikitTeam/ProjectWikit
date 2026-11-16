package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("themepreviewer", renderThemePreviewer) }

// Pages written for Wikidot swap the theme this way, so the site theme is
// dropped and the stylesheet at theme_url is imported in its place.
func renderThemePreviewer(env module.Env, params map[string]string, _ string) (string, error) {
	url := strings.TrimSpace(params["theme_url"])
	if url != "" && !stylesheetURL(url) {
		return "", &module.Error{Message: env.Text("module-themepreviewer-url", "url", url)}
	}
	if env.Page == nil {
		return "", nil
	}
	env.Page.DropTheme = max(env.Page.DropTheme, page.DropSiteTheme)
	if url != "" {
		env.Page.ComputedStyles = append(env.Page.ComputedStyles, `@import url("`+url+`");`)
	}
	return "", nil
}

func stylesheetURL(url string) bool {
	if strings.ContainsAny(url, "\"'\\<>()\n\r\t ") {
		return false
	}
	lowered := strings.ToLower(url)
	return strings.HasPrefix(lowered, "https://") || strings.HasPrefix(lowered, "http://") ||
		strings.HasPrefix(url, "//") || (strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "/\\"))
}
