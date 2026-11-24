package webapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

func (h *Articles) remove(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		body, status := csrfRefusal(r, loc)
		return body, status, nil
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

	// The source only exists until the delete below, so a notice that fails to
	// go out stops the delete rather than leaving the author with nothing.
	if err := h.noticeDeletion(ctx, loc, current, user, article); err != nil {
		return "", 0, err
	}
	public := module.Plugins != nil && module.Plugins.PagePublic(ctx, current, article.FullName())
	if err := h.deps.DB.DeleteArticle(ctx, siteID(ctx), article.ID, article.FullName()); err != nil {
		return "", 0, err
	}
	if public {
		module.Emit(current, "page.deleted", map[string]any{"page": article.FullName(), "user": usernameOf(user)})
	}
	if err := h.seen(r, user); err != nil {
		return "", 0, err
	}
	h.dropMedia(ctx, article.MediaName)
	return field("status", "ok"), http.StatusOK, nil
}

// The rows are already gone, so files that will not go are logged and left
// rather than failing a delete the reader has been told succeeded.
func (h *Articles) dropMedia(ctx context.Context, mediaName string) {
	if h.deps.Storage == nil || mediaName == "" || h.policy().KeepDeleted {
		return
	}
	store := h.backend(ctx).Store
	for _, dir := range []string{"media", "resized"} {
		prefix := storage.Join(dir, media.QuoteName(mediaName))
		if err := store.RemoveAll(ctx, prefix); err != nil {
			h.deps.log().Error("remove article media", "prefix", prefix, "err", err)
		}
	}
}

// Unless the instance keeps deleted files, a deleted attachment leaves the
// store and the table together, so its space comes back at once.
func (h *Articles) deleteFile(ctx context.Context, fileID int64, at time.Time, userID *int64) (string, bool, error) {
	if h.policy().KeepDeleted || h.deps.Storage == nil {
		return h.deps.DB.SoftDeleteFile(ctx, fileID, at, userID)
	}
	dropped, ok, err := h.deps.DB.DropFile(ctx, siteID(ctx), fileID)
	if err != nil || !ok {
		return "", ok, err
	}
	store := h.backend(ctx).Store
	stored := storage.Join(media.QuoteName(dropped.ArticleMediaName), media.QuoteName(dropped.MediaName))
	if err := store.Remove(ctx, storage.Join("media", stored)); err != nil {
		h.deps.log().Error("remove file", "key", stored, "err", err)
	}
	if err := store.RemoveAll(ctx, storage.Join("resized", stored)); err != nil {
		h.deps.log().Error("remove resized copies", "key", stored, "err", err)
	}
	return dropped.Name, true, nil
}
