package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const SubscribePath = "/pw-api/notifications/subscribe"

type Subscriptions struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Subscriptions)(nil)

func NewSubscriptions(d Deps, upstream http.Handler) *Subscriptions {
	return &Subscriptions{deps: d, upstream: upstream}
}

type subscribeRequest struct {
	PageID   string `json:"pageId"`
	ThreadID int64  `json:"forumThreadId"`
}

func (h *Subscriptions) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != SubscribePath {
		h.upstream.ServeHTTP(w, r)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		h.upstream.ServeHTTP(w, r)
		return
	}
	ctx := r.Context()
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}
	current := site.FromContext(ctx)
	if current == nil {
		h.deps.log().Error("subscribe", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-csrf-failed")))
		return
	}

	raw, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var input subscribeRequest
	if json.Unmarshal(raw, &input) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-json")))
		return
	}

	status, err := h.apply(r, user, input)
	if err != nil {
		switch {
		case errors.Is(err, errForbidden):
			writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		case errors.Is(err, errNotFound):
			writeJSON(w, http.StatusNotFound, field("error", loc.T("api-subscription-target-missing")))
		case errors.Is(err, errNoSubscription):
			writeJSON(w, http.StatusNotFound, field("error", loc.T("api-subscription-not-found")))
		case errors.Is(err, errNoTarget):
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-subscription")))
		default:
			h.deps.log().Error("subscribe", "user", user.ID, "err", err)
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		}
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "status", Value: "ok"}})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, status, body)
}

var (
	errNoTarget       = errors.New("subscription names neither a page nor a thread")
	errNoSubscription = errors.New("no such subscription")
)

func (h *Subscriptions) apply(r *http.Request, user *db.User, input subscribeRequest) (int, error) {
	ctx := r.Context()
	switch {
	case input.PageID != "":
		article, err := h.deps.DB.ArticleByName(ctx, input.PageID)
		if errors.Is(err, db.ErrNotFound) {
			return 0, errNotFound
		}
		if err != nil {
			return 0, err
		}
		if err := h.mayRead(ctx, user, article, nil); err != nil {
			return 0, err
		}
		if r.Method == http.MethodPost {
			return http.StatusOK, h.deps.DB.SubscribeToArticle(ctx, user.ID, article.ID)
		}
		gone, err := h.deps.DB.UnsubscribeFromArticle(ctx, user.ID, article.ID)
		if err != nil {
			return 0, err
		}
		if !gone {
			return 0, errNoSubscription
		}
		return http.StatusOK, nil

	case input.ThreadID > 0:
		thread, err := h.deps.DB.ForumThread(ctx, input.ThreadID)
		if errors.Is(err, db.ErrNotFound) {
			return 0, errNotFound
		}
		if err != nil {
			return 0, err
		}
		if err := h.mayRead(ctx, user, nil, thread); err != nil {
			return 0, err
		}
		if r.Method == http.MethodPost {
			return http.StatusOK, h.deps.DB.SubscribeToThread(ctx, user.ID, thread.ID)
		}
		gone, err := h.deps.DB.UnsubscribeFromThread(ctx, user.ID, thread.ID)
		if err != nil {
			return 0, err
		}
		if !gone {
			return 0, errNoSubscription
		}
		return http.StatusOK, nil
	}
	return 0, errNoTarget
}

func (h *Subscriptions) mayRead(ctx context.Context, user *db.User, article *db.Article, thread *db.ForumThread) error {
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return err
	}
	if article != nil {
		object, err := perm.Article(article, user)
		if err != nil {
			return err
		}
		if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
			return errForbidden
		}
		return nil
	}
	object, err := perm.ForumThread(thread, user)
	if err != nil {
		return err
	}
	if !perms.Resolve(subject, object).Has(perms.ViewForumThreads) {
		return errForbidden
	}
	return nil
}
