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

func renderCSS(env module.Env, params map[string]string, body string) (string, error) {
	head := module.BoolParam(pathUnder(env, params), "head", false)

	source := strings.ReplaceAll(body, nbsp, " ")
	minified, err := cssMinifier.String("text/css", source)
	if err != nil {
		minified = source
	}
	minified = strings.ReplaceAll(minified, "<", lessThan)

	if env.Page != nil {
		env.Page.AddCSS += source + "\n"
		if head {
			env.Page.ComputedStyle += minified
		}
	}

	if head {
		return "\n" + ind8, nil
	}
	return "\n" + ind12 + `<style>` + minified + `</style>` +
		"\n" + ind8 + "\n" + ind8, nil
}
