package webapi

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

// Every log entry a save writes is announced on its own, so an edit that both
// renames and retitles arrives as two.
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
	if err := h.logEdit(r, article, entry, editor); err != nil {
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

// The revision that created the page is left out, since the page's own entry
// already says it was created.
func (h *Articles) logEdit(r *http.Request, article *db.Article, entry db.LogEntry, editor *db.User) error {
	if entry.Type == db.LogNew {
		return nil
	}
	meta, err := decodeMeta(entry.Meta)
	if err != nil {
		return err
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "article", Value: article.FullName()},
		{Key: "comment", Value: entry.Comment},
		{Key: "edit_type", Value: entry.Type},
		{Key: "rev_number", Value: entry.RevNumber},
		{Key: "log_entry_meta", Value: fromJSON(meta)},
	})
	if err != nil {
		return err
	}
	return h.act(r, editor, db.ActionEditArticle, body)
}

func (h *Articles) logCreate(r *http.Request, article *db.Article, editor *db.User) error {
	body, err := json.Marshal(map[string]any{"article": article.FullName()})
	if err != nil {
		return err
	}
	return h.act(r, editor, db.ActionCreateArticle, string(body))
}

func (h *Articles) logDelete(r *http.Request, article *db.Article, editor *db.User, rating page.Rating) error {
	body, err := json.Marshal(map[string]any{
		"article":    article.FullName(),
		"rating":     rating.Value,
		"votes":      rating.Votes,
		"popularity": rating.Popularity,
	})
	if err != nil {
		return err
	}
	return h.act(r, editor, db.ActionRemoveArticle, string(body))
}

// The address is the one the entry layer trusted, not whatever the request
// claimed, so a reverse proxy cannot be talked into logging a made-up client.
func (h *Articles) act(r *http.Request, user *db.User, kind, meta string) error {
	var id *int64
	name := ""
	if user != nil {
		id, name = &user.ID, user.Username
	}
	var ip *netip.Addr
	if h.deps.Trust != nil {
		if addr, ok := h.deps.Trust.ClientIP(r); ok {
			ip = &addr
		}
	}
	return h.deps.DB.AddActionLog(r.Context(), id, name, kind, meta, ip, time.Now().UTC())
}
