package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("button", renderButton) }

func renderButton(env module.Env, params map[string]string, _ string) (string, error) {
	kind := strings.ToLower(strings.TrimSpace(params["type"]))
	if kind != "edit" {
		return "", &module.Error{Message: env.Text("module-button-unknown", "type", params["type"])}
	}
	if env.Page == nil || env.Page.Article == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}

	label, ok := params["text"]
	if !ok {
		label = env.Text("module-button-edit")
	}
	// The path parameter is what opens the editor on arrival, and it is the
	// same one the missing-page view already answers to.
	href := "/" + env.Page.Article.FullName() + "/edit/true"
	return `<a class="button" href="` + escape.HTML(href) + `">` + escape.HTML(label) + "</a>", nil
}
