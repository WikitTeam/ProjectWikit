package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("button", renderButton) }

// Escaping for JavaScript first lets the HTML escape that follows keep a quote
// inside the string rather than ending the attribute.
var jsString = strings.NewReplacer(`\`, `\\`, `'`, `\'`)

func renderButton(env module.Env, params map[string]string, _ string) (string, error) {
	kind := strings.ToLower(strings.TrimSpace(params["type"]))

	var call string
	switch kind {
	case "edit":
		call = "pwikit.edit(event)"
	case "set-tags":
		call = "pwikit.setTags(event, '" + jsString.Replace(params["tags"]) + "')"
	default:
		return "", &module.Error{Message: env.Text("module-button-unknown", "type", params["type"])}
	}
	if env.Page == nil || env.Page.Article == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}

	// Wikidot trims the label, and a kept trailing space would sit inside the
	// button's own box and swallow the gap to the next one.
	label, ok := params["text"]
	if !ok {
		label = params["type"]
	}
	label = strings.TrimSpace(label)

	// Wikidot gives every kind the same class, so data-button-type is the only
	// thing a stylesheet has to tell one button from another.
	return `<a class="wiki-standalone-button" data-button-type="` + escape.HTML(kind) +
		`" href="javascript:;" onclick="` + escape.HTML(call) + `">` +
		escape.HTML(label) + `</a>`, nil
}
