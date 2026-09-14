package localitem

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func codeMime(language string) string {
	switch strings.ToLower(language) {
	case "html", "xhtml":
		return "text/html; charset=utf-8"
	case "javascript", "js", "jsx":
		return "text/javascript; charset=utf-8"
	case "xml":
		return "application/xml; charset=utf-8"
	case "css":
		return "text/css; charset=utf-8"
	}
	return "text/plain; charset=utf-8"
}

func (h *handler) code(req *request, rest string) (item, error) {
	index, err := strconv.Atoi(rest)
	if err != nil {
		return missing(noCode), nil
	}

	parts, err := h.parts(req, renderer.ModeSystem)
	if err != nil {
		return item{}, err
	}

	index--
	if index < 0 || index >= len(parts.Code) {
		return missing(noCode), nil
	}
	block := parts.Code[index]
	return found(codeMime(block.Language), block.Source), nil
}

// Page variables belong to a rendered page, and a code block hands back what
// its author typed, so the source goes in as it was written.
func (h *handler) parts(req *request, mode renderer.Mode) (renderer.Parts, error) {
	info, err := req.env.PageInfo(req.article)
	if err != nil {
		return renderer.Parts{}, err
	}
	vars := req.env.Vars(req.article)
	pc := page.NewContext(req.article, req.article, req.params, req.user)
	return req.env.CodeAndHTML(req.source, info, req.env.Callbacks(vars, pc), mode)
}
