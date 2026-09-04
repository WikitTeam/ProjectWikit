package webapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
)

func (h *Articles) fetch(r *http.Request, _ *i18n.Localizer, name string) (string, int, error) {
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	source, err := h.latestSource(r.Context(), article.ID)
	if err != nil {
		return "", 0, err
	}
	body, err := h.articleJSON(r, article, source)
	return body, http.StatusOK, err
}

// A page with no version behind it still answers, with a null where its source
// would be, which is not the same as a page whose source is empty.
func (h *Articles) latestSource(ctx context.Context, articleID int64) (*string, error) {
	source, err := h.deps.DB.LatestSource(ctx, articleID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &source, nil
}
