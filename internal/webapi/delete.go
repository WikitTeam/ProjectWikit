package webapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

func (h *Articles) remove(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		return field("error", loc.T("api-csrf-failed")), http.StatusForbidden, nil
	}

	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	user := auth.FromContext(ctx)
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return "", 0, err
	}
	object, err := perm.Article(article, user)
	if err != nil {
		return "", 0, err
	}
	if !perms.Resolve(subject, object).Has(perms.DeleteArticles) {
		return "", 0, errForbidden
	}

	rating, err := h.rating(r, article)
	if err != nil {
		return "", 0, err
	}
	if err := h.deps.DB.DeleteArticle(ctx, article.ID, article.FullName()); err != nil {
		return "", 0, err
	}
	if err := h.logDelete(r, article, user, rating); err != nil {
		return "", 0, err
	}
	h.dropMedia(article.MediaName)
	return field("status", "ok"), http.StatusOK, nil
}

// The rows are already gone, so a directory that will not go is logged and left
// rather than failing a delete the reader has been told succeeded.
func (h *Articles) dropMedia(mediaName string) {
	if h.deps.Files == "" || mediaName == "" {
		return
	}
	dir := filepath.Join(h.deps.Files, "media", mediaName)
	if err := os.RemoveAll(dir); err != nil {
		h.deps.log().Error("remove article media", "dir", dir, "err", err)
	}
}
