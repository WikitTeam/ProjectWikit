package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("pageimage", renderPageImage) }

func renderPageImage(env module.Env, params map[string]string, _ string) (string, error) {
	src, ok := params["src"]
	if !ok {
		return "", nil
	}
	if env.Page != nil {
		env.Page.OGImage = resourceURL(src, env.MediaDomain())
	}
	return "", nil
}

// resourceURL turns what an author wrote into a link the browser can follow. A
// name with no slash in it belongs to the page being rendered, so it cannot be
// resolved here and is handed back untouched.
func resourceURL(raw, mediaDomain string) string {
	uri, err := validateURL(raw)
	if err != nil {
		uri = "#invalid-url"
	}
	if uri == "" {
		return ""
	}
	if strings.Contains(uri, "//") {
		return uri
	}
	uri = strings.TrimPrefix(uri, "/")
	if !strings.Contains(uri, "/") {
		return uri
	}
	return "https://" + mediaDomain + "/local--files/" + uri
}
