package webapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type contentReportRequest struct {
	UserID int64  `json:"user_id"`
	Page   string `json:"page"`
	PostID int64  `json:"post_id"`
	Reason string `json:"reason"`
}

const reportExcerpt = 500

// A page or a post is stored the way reported messages are, one record naming
// who and what, so the report screen reads every kind the same way.
func (h *Messages) reportContent(r *http.Request, loc *i18n.Localizer, user *db.User) (string, int, error) {
	ctx := r.Context()
	raw, err := readBody(r)
	if err != nil {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-bad-request"))
	}
	var input contentReportRequest
	if json.Unmarshal(raw, &input) != nil {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-bad-json"))
	}
	reason := strings.TrimSpace(input.Reason)
	switch {
	case reason == "":
		return "", 0, deny(http.StatusBadRequest, loc.T("api-missing-reason"))
	case utf8.RuneCountInString(reason) > maxReportReason:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-reason-too-long", "max", maxReportReason))
	}

	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return "", 0, err
	}
	var reportedID int64
	var record wikijson.Object
	switch {
	case input.UserID != 0:
		reported, err := h.deps.DB.UserByID(ctx, input.UserID)
		if errors.Is(err, db.ErrNotFound) {
			return "", 0, errNotFound
		} else if err != nil {
			return "", 0, err
		}
		reportedID = reported.ID
		record = reportRecord(reported.DisplayLabel(), loc.T("report-user-line", "url", "/-/users/"+reported.Username), time.Now())
	case input.Page != "":
		article, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), wikidot.Normalize(input.Page))
		if errors.Is(err, db.ErrNotFound) {
			return "", 0, errNotFound
		} else if err != nil {
			return "", 0, err
		}
		object, err := perm.Article(article, user)
		if err != nil {
			return "", 0, err
		}
		if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
			return "", 0, errNotFound
		}
		authors, err := h.deps.DB.ArticleAuthors(ctx, article.ID)
		if err != nil {
			return "", 0, err
		}
		if len(authors) == 0 {
			return "", 0, deny(http.StatusBadRequest, loc.T("api-missing-reported"))
		}
		reportedID = authors[0].ID
		record = reportRecord(authors[0].DisplayLabel(),
			loc.T("report-page-line", "title", article.DisplayName(), "url", "/"+article.FullName()), article.UpdatedAt)
	case input.PostID != 0:
		post, err := h.deps.DB.ForumPost(ctx, siteID(ctx), input.PostID)
		if errors.Is(err, db.ErrNotFound) || (err == nil && post.AuthorID == nil) {
			return "", 0, errNotFound
		} else if err != nil {
			return "", 0, err
		}
		thread, err := h.deps.DB.ForumThread(ctx, siteID(ctx), post.ThreadID)
		if errors.Is(err, db.ErrNotFound) {
			return "", 0, errNotFound
		} else if err != nil {
			return "", 0, err
		}
		visible, err := postVisible(r, perm, subject, user, thread, h.deps.DB)
		if err != nil {
			return "", 0, err
		}
		if !visible {
			return "", 0, errNotFound
		}
		author, err := h.deps.DB.UserByID(ctx, *post.AuthorID)
		if err != nil {
			return "", 0, err
		}
		source, err := h.deps.DB.ForumPostSource(ctx, post.ID, nil)
		if err != nil {
			return "", 0, err
		}
		if runes := []rune(source); len(runes) > reportExcerpt {
			source = string(runes[:reportExcerpt]) + "…"
		}
		url := "/forum/t-" + strconv.FormatInt(post.ThreadID, 10) + "#post-" + strconv.FormatInt(post.ID, 10)
		reportedID = author.ID
		record = reportRecord(author.DisplayLabel(), loc.T("report-post-line", "url", url)+"\n"+source, post.CreatedAt)
	default:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-missing-reported"))
	}
	recent, err := h.deps.DB.ReportsSince(ctx, siteID(ctx), user.ID, reportedID, time.Now().Add(-reportRateLimitScope))
	if err != nil {
		return "", 0, err
	}
	if recent >= maxReportsPerDay {
		return "", 0, deny(http.StatusTooManyRequests, loc.T("api-report-rate-limited"))
	}
	snapshot, err := wikijson.Marshal(wikijson.Array{record})
	if err != nil {
		return "", 0, err
	}
	reported, err := h.deps.DB.UserByID(ctx, reportedID)
	if err != nil {
		return "", 0, err
	}
	now := time.Now()
	id, err := h.deps.DB.CreateReport(ctx, siteID(ctx), user.ID, reportedID, reason, snapshot, now)
	if err != nil {
		return "", 0, err
	}
	if err := h.tellReviewers(r, id, user, reported, now); err != nil {
		h.deps.log().Error("announce report", "report", id, "err", err)
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "status", Value: "ok"}, {Key: "report_id", Value: id}})
	return body, http.StatusOK, err
}

func reportRecord(name, body string, at time.Time) wikijson.Object {
	return wikijson.Object{
		{Key: "sender_name", Value: name},
		{Key: "body", Value: body},
		{Key: "created_at", Value: at.UTC().Format(time.RFC3339)},
	}
}

// A post in a page's discussion follows the page, the rest follow the forum.
func postVisible(r *http.Request, perm *repo.Perms, subject perms.Subject, user *db.User, thread *db.ForumThread, d *db.DB) (bool, error) {
	if thread.ArticleID == nil {
		granted := perms.Resolve(subject, nil)
		return granted.Has(perms.ViewForumThreads) && granted.Has(perms.ViewForumPosts), nil
	}
	article, err := d.ArticleByID(r.Context(), siteID(r.Context()), *thread.ArticleID)
	if errors.Is(err, db.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	object, err := perm.Article(article, user)
	if err != nil {
		return false, err
	}
	granted := perms.Resolve(subject, object)
	return granted.Has(perms.ViewArticles) && granted.Has(perms.ViewArticleComments), nil
}
