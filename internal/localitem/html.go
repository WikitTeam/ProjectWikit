package localitem

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func (h *handler) html(req *request, rest string) (item, error) {
	hash, id, ok := strings.Cut(rest, "-")
	if !ok {
		return missing(noHTML), nil
	}

	info, err := req.env.PageInfo(req.article)
	if err != nil {
		return item{}, err
	}
	vars := req.env.Vars(req.article)
	pc := page.NewContext(req.article, req.article, req.params, req.user)
	cb := req.env.Callbacks(vars, pc)

	// A block can live on an included page. The text render is the pass that
	// pulls those in, so the blocks are collected off it.
	result, err := req.env.Text(req.source, info, cb, renderer.ModeSystem)
	if err != nil {
		return item{}, err
	}

	block, ok := blockByHash(result.HTML, hash)
	if !ok {
		return missing(noHTML), nil
	}
	prepend, err := cb.GetHTMLInjectedCode(id)
	if err != nil {
		return item{}, err
	}
	return found(htmlMime, prepend+block), nil
}

func blockByHash(blocks []string, hash string) (string, bool) {
	for _, block := range blocks {
		sum := md5.Sum([]byte(block))
		if hex.EncodeToString(sum[:]) == hash {
			return block, true
		}
	}
	return "", false
}
