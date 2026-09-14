package webapi

import (
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func (h *Articles) notifyRevision(r *http.Request, article *db.Article, rev db.Revision) error {
	ctx := r.Context()
	if rev.EntryID == 0 {
		return nil
	}
	entry, err := h.deps.DB.LogEntryByID(ctx, rev.EntryID)
	if err != nil {
		return err
	}
	editor := auth.FromContext(ctx)
	if err := h.seen(r, editor); err != nil {
		return err
	}
	readers, err := h.revisionReaders(r, article, editor)
	if err != nil || len(readers) == 0 {
		return err
	}

	who, err := repo.UserJSON(ctx, h.deps.DB, editor)
	if err != nil {
		return err
	}
	meta, err := decodeMeta(entry.Meta)
	if err != nil {
		return err
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "user", Value: who},
		{Key: "article", Value: wikijson.Object{
			{Key: "uid", Value: article.ID},
			{Key: "pageId", Value: article.FullName()},
			{Key: "title", Value: article.Title},
		}},
		{Key: "rev_id", Value: rev.EntryID},
		{Key: "rev_meta", Value: fromJSON(meta)},
		{Key: "rev_number", Value: entry.RevNumber},
		{Key: "rev_type", Value: entry.Type},
		{Key: "comment", Value: entry.Comment},
	})
	if err != nil {
		return err
	}
	return h.deps.DB.SendNotification(ctx, db.NotifyNewArticleRevision, body, readers, time.Now().UTC())
}

func (h *Articles) revisionReaders(r *http.Request, article *db.Article, editor *db.User) ([]int64, error) {
	ctx := r.Context()
	subscribers, err := h.deps.DB.ArticleSubscribers(ctx, article.ID)
	if err != nil {
		return nil, err
	}
	perm := repo.NewPerms(ctx, h.deps.DB)
	at := time.Now()

	out := make([]int64, 0, len(subscribers))
	for _, id := range subscribers {
		if editor != nil && id == editor.ID {
			continue
		}
		reader, err := h.deps.DB.UserByID(ctx, id)
		if err != nil {
			return nil, err
		}
		subject, err := perm.Subject(reader, at)
		if err != nil {
			return nil, err
		}
		object, err := perm.Article(article, reader)
		if err != nil {
			return nil, err
		}
		if perms.Resolve(subject, object).Has(perms.ViewArticles) {
			out = append(out, id)
		}
	}
	return out, nil
}

func (h *Articles) seen(r *http.Request, user *db.User) error {
	if user == nil || h.deps.Trust == nil {
		return nil
	}
	addr, ok := h.deps.Trust.ClientIP(r)
	if !ok {
		return nil
	}
	return h.deps.DB.SeenAddress(r.Context(), user.ID, &addr, time.Now().UTC())
}
