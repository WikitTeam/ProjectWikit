package localitem

import (
	"net/http"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

const themeFile = "style.css"

const (
	noIncludeOpen  = "[[noinclude]]"
	noIncludeClose = "[[/noinclude]]"
)

// The parameters come from the query string, so anyone can make one value
// repeat across a page that names it many times.
const maxParamGrowth = 1 << 20

func (h *handler) theme(req *request, rest string) (item, error) {
	if rest != themeFile {
		return missing(noResource), nil
	}

	source, ok := expandParams(stripNoInclude(req.source), stringMap(req.query.Get("includeParams")))
	if !ok {
		return item{status: http.StatusRequestEntityTooLarge, contentType: textMime,
			body: http.StatusText(http.StatusRequestEntityTooLarge)}, nil
	}

	info, err := req.env.PageInfo(req.article)
	if err != nil {
		return item{}, err
	}
	vars := req.env.Vars(req.article)
	pc := page.NewContext(req.article, req.article, req.params, req.user)
	if _, err := req.env.HTML(page.PreRender(source, vars), info, req.env.Callbacks(vars, pc), renderer.ModeArticle); err != nil {
		return item{}, err
	}
	return found(cssMime, pc.AddCSS), nil
}

// An opening tag with nothing closing it takes the rest of the source with it,
// since what follows was written to stay out of an include either way.
func stripNoInclude(source string) string {
	var out strings.Builder
	for {
		start := strings.Index(source, noIncludeOpen)
		if start < 0 {
			if out.Len() == 0 {
				return source
			}
			out.WriteString(source)
			return out.String()
		}
		out.WriteString(source[:start])
		end := strings.Index(source[start:], noIncludeClose)
		if end < 0 {
			return out.String()
		}
		source = source[start+end+len(noIncludeClose):]
	}
}

// One pass, so a value that spells another parameter stays as written instead
// of being expanded again.
func expandParams(source string, params map[string]string) (string, bool) {
	pairs := make([]string, 0, 2*len(params))
	growth := 0
	for key, value := range params {
		token := "{$" + key + "}"
		n := strings.Count(source, token)
		if n == 0 {
			continue
		}
		pairs = append(pairs, token, value)
		if extra := len(value) - len(token); extra > 0 {
			growth += n * extra
			if growth > maxParamGrowth {
				return "", false
			}
		}
	}
	if len(pairs) == 0 {
		return source, true
	}
	return strings.NewReplacer(pairs...).Replace(source), true
}
