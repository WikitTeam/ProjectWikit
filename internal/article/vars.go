package article

import (
	"fmt"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
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
				return jsonString(literal(name)), true
			case param.Bare:
				return "null", true
			}
			return jsonString(param.Value), true

		case strings.HasPrefix(name, prefixURL):
			param, ok := params.Lookup(lookupKey(name, prefixURL))
			if !ok {
				return page.QuoteAll(literal(name)), true
			}
			// A bare key crashes Django here, and the empty string is what the
			// rest of this family answers for one.
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

// jsonString spells a string the way Python's json.dumps does, which escapes
// every character outside printable ASCII and leaves the HTML ones alone.
func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r >= 0x20 && r <= 0x7e:
				b.WriteRune(r)
			case r > 0xffff:
				r -= 0x10000
				fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
			default:
				fmt.Fprintf(&b, `\u%04x`, r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
