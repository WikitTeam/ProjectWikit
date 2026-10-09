package webapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	RatingsPath    = "/pw-api/ratings"
	LikedPostsPath = "/pw-api/liked-posts"
	MyTicketsPath  = "/pw-api/my-tickets"
	MyTicketPrefix = MyTicketsPath + "/"
)

const ownRowsPerPage = 20

type OwnRows struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*OwnRows)(nil)

func NewOwnRows(d Deps, next http.Handler) *OwnRows {
	return &OwnRows{deps: d, next: next}
}

func (h *OwnRows) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	detail := strings.HasPrefix(r.URL.Path, MyTicketPrefix)
	if r.Method == http.MethodDelete && r.URL.Path == MyTicketsPath {
		h.clearTickets(w, r)
		return
	}
	if r.Method != http.MethodGet || (r.URL.Path != RatingsPath && r.URL.Path != LikedPostsPath && r.URL.Path != MyTicketsPath && !detail) {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}
	if r.URL.Path == RatingsPath {
		h.ratings(w, r, loc, user)
		return
	}
	if r.URL.Path == MyTicketsPath {
		h.tickets(w, r, loc, user)
		return
	}
	if detail {
		h.ticket(w, r, loc, user)
		return
	}
	h.likedPosts(w, r, loc, user)
}

func (h *OwnRows) ratings(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	total, err := h.deps.DB.RatedByCountOf(ctx, user.ID)
	if err != nil {
		h.deps.log().Error("count ratings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	page, pages := pageOf(r, total)

	found, err := h.deps.DB.RatedByOnEverySite(ctx, user.ID, (page-1)*ownRowsPerPage, ownRowsPerPage)
	if err != nil {
		h.deps.log().Error("list ratings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	sites, err := loadSiteLinks(ctx, h.deps.DB)
	if err != nil {
		h.deps.log().Error("list sites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		title := one.Article.Title
		if title == "" {
			title = one.Article.FullName()
		}
		votedAt := any(nil)
		if one.VotedAt != nil {
			votedAt = isoTime(*one.VotedAt)
		}
		rendered = append(rendered, wikijson.Object{
			{Key: "pageId", Value: one.Article.FullName()},
			{Key: "site", Value: sites.title(one.SiteID)},
			{Key: "url", Value: sites.href(one.SiteID, "/"+one.Article.FullName())},
			{Key: "title", Value: title},
			{Key: "rate", Value: one.Rate},
			{Key: "votedAt", Value: votedAt},
		})
	}
	h.writePage(w, loc, page, pages, total, "ratings", rendered)
}

func (h *OwnRows) likedPosts(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	total, err := h.deps.DB.LikedPostCountOf(ctx, user.ID)
	if err != nil {
		h.deps.log().Error("count liked posts", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	page, pages := pageOf(r, total)

	found, err := h.deps.DB.LikedPostsOf(ctx, user.ID, (page-1)*ownRowsPerPage, ownRowsPerPage)
	if err != nil {
		h.deps.log().Error("list liked posts", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	sites, err := loadSiteLinks(ctx, h.deps.DB)
	if err != nil {
		h.deps.log().Error("list sites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		thread := "/forum/t-" + strconv.FormatInt(one.Post.ThreadID, 10) + "/" + wikidot.Normalize(one.ThreadName)
		rendered = append(rendered, wikijson.Object{
			{Key: "postId", Value: one.Post.ID},
			{Key: "name", Value: strings.TrimSpace(one.Post.Name)},
			{Key: "threadName", Value: one.ThreadName},
			{Key: "site", Value: sites.title(one.SiteID)},
			{Key: "url", Value: sites.href(one.SiteID, thread+"#post-"+strconv.FormatInt(one.Post.ID, 10))},
			{Key: "likedAt", Value: isoTime(one.LikedAt)},
		})
	}
	h.writePage(w, loc, page, pages, total, "posts", rendered)
}

func (h *OwnRows) tickets(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	total, err := h.deps.DB.OwnTicketCount(ctx, user.ID)
	if err != nil {
		h.deps.log().Error("count own tickets", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	page, pages := pageOf(r, total)

	found, err := h.deps.DB.OwnTickets(ctx, user.ID, (page-1)*ownRowsPerPage, ownRowsPerPage)
	if err != nil {
		h.deps.log().Error("list own tickets", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	sites, err := loadSiteLinks(ctx, h.deps.DB)
	if err != nil {
		h.deps.log().Error("list sites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		reviewedAt := any(nil)
		if one.ReviewedAt != nil {
			reviewedAt = isoTime(*one.ReviewedAt)
		}
		rendered = append(rendered, wikijson.Object{
			{Key: "kind", Value: one.Kind},
			{Key: "id", Value: one.ID},
			{Key: "site", Value: sites.title(one.SiteID)},
			{Key: "url", Value: sites.href(one.SiteID, "/")},
			{Key: "subject", Value: one.Subject},
			{Key: "status", Value: one.Status},
			{Key: "reply", Value: one.Reply},
			{Key: "createdAt", Value: isoTime(one.CreatedAt)},
			{Key: "reviewedAt", Value: reviewedAt},
		})
	}
	h.writePage(w, loc, page, pages, total, "tickets", rendered)
}

func (h *OwnRows) ticket(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	kind, id, ok := ownTicketRef(strings.TrimPrefix(r.URL.Path, MyTicketPrefix))
	if !ok {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-ticket-not-found")))
		return
	}
	found, err := h.deps.DB.OwnTicketDetail(ctx, user.ID, kind, id)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-ticket-not-found")))
		return
	}
	if err != nil {
		h.deps.log().Error("read own ticket", "kind", kind, "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	sites, err := loadSiteLinks(ctx, h.deps.DB)
	if err != nil {
		h.deps.log().Error("list sites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	reviewedAt := any(nil)
	if found.ReviewedAt != nil {
		reviewedAt = isoTime(*found.ReviewedAt)
	}
	sourceURL := ""
	if found.SourcePage != "" {
		sourceURL = sites.href(found.SiteID, "/"+found.SourcePage)
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "kind", Value: found.Kind},
		{Key: "id", Value: found.ID},
		{Key: "site", Value: sites.title(found.SiteID)},
		{Key: "url", Value: sites.href(found.SiteID, "/")},
		{Key: "subject", Value: found.Subject},
		{Key: "status", Value: found.Status},
		{Key: "reply", Value: found.Reply},
		{Key: "createdAt", Value: isoTime(found.CreatedAt)},
		{Key: "reviewedAt", Value: reviewedAt},
		{Key: "body", Value: found.Body},
		{Key: "sourcePage", Value: found.SourcePage},
		{Key: "sourceUrl", Value: sourceURL},
		{Key: "messages", Value: reportedMessages(found.Messages)},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type clearTicketsRequest struct {
	All   bool `json:"all"`
	Items []struct {
		Kind string `json:"kind"`
		ID   int64  `json:"id"`
	} `json:"items"`
}

func (h *OwnRows) clearTickets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loc := h.deps.Bundle.For(ctx)
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}
	raw, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var input clearTicketsRequest
	if json.Unmarshal(raw, &input) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-json")))
		return
	}
	refs := make([]db.OwnTicketRef, 0, len(input.Items))
	for _, one := range input.Items {
		kind, id, ok := ownTicketRef(one.Kind + "/" + strconv.FormatInt(one.ID, 10))
		if !ok {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		refs = append(refs, db.OwnTicketRef{Kind: kind, ID: id})
	}
	if !input.All && len(refs) == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var hidden int64
	if input.All {
		hidden, err = h.deps.DB.HideAllOwnTickets(ctx, user.ID, time.Now().UTC())
	} else {
		hidden, err = h.deps.DB.HideOwnTickets(ctx, user.ID, refs, time.Now().UTC())
	}
	if err != nil {
		h.deps.log().Error("hide own tickets", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "hidden", Value: hidden}})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func ownTicketRef(rest string) (string, int64, bool) {
	kind, raw, ok := strings.Cut(rest, "/")
	if !ok {
		return "", 0, false
	}
	switch kind {
	case db.TicketKind, db.MembershipApplyKind, db.ReportKind:
	default:
		return "", 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false
	}
	return kind, id, true
}

func reportedMessages(raw string) wikijson.Array {
	var found []struct {
		SenderName string    `json:"sender_name"`
		Body       string    `json:"body"`
		CreatedAt  time.Time `json:"created_at"`
	}
	out := wikijson.Array{}
	if raw == "" || json.Unmarshal([]byte(raw), &found) != nil {
		return out
	}
	for _, one := range found {
		at := any(nil)
		if !one.CreatedAt.IsZero() {
			at = isoTime(one.CreatedAt)
		}
		out = append(out, wikijson.Object{
			{Key: "sender", Value: one.SenderName},
			{Key: "body", Value: one.Body},
			{Key: "createdAt", Value: at},
		})
	}
	return out
}

func (h *OwnRows) writePage(w http.ResponseWriter, loc *i18n.Localizer, page, pages, total int, key string, rows wikijson.Array) {
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "page", Value: page},
		{Key: "pages", Value: pages},
		{Key: "total", Value: total},
		{Key: key, Value: rows},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func pageOf(r *http.Request, total int) (page, pages int) {
	pages = (total + ownRowsPerPage - 1) / ownRowsPerPage
	if pages == 0 {
		pages = 1
	}
	page = 1
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		page = n
	}
	if page > pages {
		page = pages
	}
	return page, pages
}
