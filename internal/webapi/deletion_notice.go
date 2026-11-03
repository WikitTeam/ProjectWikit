package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

// The title and the source are the author's own text, so the parser only ever
// sees these tokens. The escaped text replaces them once the template is HTML.
const (
	titleToken  = "pwikitdeletedtitle7f3a"
	sourceToken = "pwikitdeletedsource7f3a"
)

func (h *Articles) noticeDeletion(ctx context.Context, loc *i18n.Localizer, current *db.Site, deleter *db.User, article *db.Article) error {
	if !current.DeletionNoticeEnabled {
		return nil
	}
	authors, err := h.deps.DB.ArticleAuthors(ctx, article.ID)
	if err != nil || len(authors) == 0 {
		return err
	}
	source, err := h.deps.DB.LatestSource(ctx, article.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return err
	}

	template := strings.TrimSpace(current.DeletionNotice)
	if template == "" {
		template = loc.T("deletion-notice-default")
	}
	name := article.FullName()
	wikitext := strings.NewReplacer(
		"{pagename}", name,
		"{pageurl}", name,
		"{pagetitle}", titleToken,
		"{source}", sourceToken,
	).Replace(template)

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons, Trust: h.deps.Trust}.Env(ctx, loc, current, deleter)
	info, err := env.PageInfo(nil)
	if err != nil {
		return err
	}
	out, err := env.HTML(wikitext, info, env.Callbacks(nil, page.NewContext(nil, nil, nil, nil)), renderer.ModeSystem)
	if err != nil {
		return err
	}
	message := strings.NewReplacer(
		titleToken, escape.HTML(article.Title),
		sourceToken, escape.HTML(source),
	).Replace(out.Body)

	meta, err := json.Marshal(map[string]string{"message": message, "title": article.Title, "page": name})
	if err != nil {
		return err
	}
	ids := make([]int64, len(authors))
	for i, author := range authors {
		ids[i] = author.ID
	}
	return h.deps.DB.SendNotification(ctx, current.ID, db.NotifyPageDeleted, string(meta), ids, time.Now().UTC())
}
