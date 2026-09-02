package localitem

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

const themeFile = "style.css"

const (
	noIncludeOpen  = "[[noinclude]]"
	noIncludeClose = "[[/noinclude]]"
)

func (h *handler) theme(req *request, rest string) (item, error) {
	if rest != themeFile {
		return missing(noResource), nil
	}

	source := stripNoInclude(req.source)
	for key, value := range stringMap(req.query.Get("includeParams")) {
		source = strings.ReplaceAll(source, "{$"+key+"}", value)
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
	for {
		start := strings.Index(source, noIncludeOpen)
		if start < 0 {
			return source
		}
		end := strings.Index(source[start:], noIncludeClose)
		if end < 0 {
			return source[:start]
		}
		source = source[:start] + source[start+end+len(noIncludeClose):]
	}
}
