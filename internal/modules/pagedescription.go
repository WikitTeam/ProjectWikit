package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("pagedescription", renderPageDescription) }

const (
	nbsp = string(rune(0xa0))
	// The description reaches a JSON string, where a raw < would close the
	// script tag that carries it.
	lessThan = `\u003c`
)

func renderPageDescription(env module.Env, _ map[string]string, body string) (string, error) {
	text := strings.ReplaceAll(body, nbsp, " ")
	text = strings.ReplaceAll(text, "<", lessThan)
	if text != "" && env.Page != nil {
		env.Page.OGDescription = text
	}
	return "", nil
}
