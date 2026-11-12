package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("notheme", renderNoTheme) }

func renderNoTheme(env module.Env, params map[string]string, _ string) (string, error) {
	drop := page.DropSiteTheme
	switch strings.ToLower(strings.TrimSpace(params["scope"])) {
	case "", "site":
	case "all":
		drop = page.DropAllThemes
	default:
		return "", &module.Error{Message: env.Text("module-notheme-scope", "scope", params["scope"])}
	}
	if env.Page != nil {
		env.Page.DropTheme = max(env.Page.DropTheme, drop)
	}
	return "", nil
}
