package article

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	prefixPath = "path|"
	prefixExpr = "path_expr|"
	prefixURL  = "path_url|"
	varCanonic = "canonical_url"
)

// ThisPage answers the substitutions only the request can answer. The name it
// is handed keeps its case while the key it looks up does not, so %%PATH|a%%
// resolves to nothing at all.
func ThisPage(params Params, canonicalURL string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		switch {
		case strings.HasPrefix(name, prefixExpr):
			param, ok := params.Lookup(lookupKey(name, prefixExpr))
			switch {
			case !ok:
				return wikijson.String(literal(name)), true
			case param.Bare:
				return "null", true
			}
			return wikijson.String(param.Value), true

		case strings.HasPrefix(name, prefixURL):
			param, ok := params.Lookup(lookupKey(name, prefixURL))
			if !ok {
				return page.QuoteAll(literal(name)), true
			}
			// The empty string is what the rest of this family answers for a bare key.
			return page.QuoteAll(param.Value), true

		case strings.HasPrefix(name, prefixPath):
			param, ok := params.Lookup(lookupKey(name, prefixPath))
			if !ok || param.Bare {
				return "", false
			}
			return param.Value, true

		case name == varCanonic:
			return canonicalURL, true
		}
		return "", false
	}
}

func lookupKey(name, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(name, prefix))
}

func literal(name string) string { return "%%" + name + "%%" }
