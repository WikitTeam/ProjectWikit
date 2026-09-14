package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

func init() { module.Register("css", renderCSS) }

var cssMinifier = newCSSMinifier()

func newCSSMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	return m
}

// Every block goes to the head, so a rule cannot lose to one that merely sits
// higher up the page. The head argument is accepted and ignored.
func renderCSS(env module.Env, params map[string]string, body string) (string, error) {
	source := strings.ReplaceAll(body, nbsp, " ")
	minified, err := cssMinifier.String("text/css", source)
	if err != nil {
		minified = source
	}
	minified = strings.ReplaceAll(minified, "<", lessThan)

	if env.Page != nil {
		env.Page.AddCSS += source + "\n"
		env.Page.ComputedStyle += minified
	}

	return "\n" + ind8, nil
}
