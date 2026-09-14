package modules

import (
	"net/url"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("redirect", renderRedirect) }

func renderRedirect(env module.Env, params map[string]string, _ string) (string, error) {
	params = pathUnder(env, params)
	if module.BoolParam(params, "noredirect", false) {
		return "", nil
	}
	target, err := validateURL(params["destination"])
	if err != nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "redirect")}
	}
	if env.Page != nil {
		env.Page.RedirectTo = target
	}
	return "", nil
}

// validateURL keeps out the two schemes that would run as soon as a reader
// followed the link. Everything else, valid or not, is passed through.
func validateURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "javascript", "data":
		return "", errURLScheme
	}
	return trimmed, nil
}

var errURLScheme = &module.Error{Message: "url scheme is not allowed"}
