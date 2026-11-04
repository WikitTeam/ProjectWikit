package webapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	NotificationsPath     = "/pw-api/notifications"
	NotificationsMutePath = "/pw-api/notifications/mute"
)

const defaultNotificationPage = 10

// The reader may ask for one kind at a time. Anything the catalogue does not
// know is refused rather than quietly widened to everything.
var notificationKinds = []string{
	db.NotifyWelcome,
	db.NotifyExternal,
	db.NotifyNewPostReply,
	db.NotifyNewThreadPost,
	db.NotifyNewArticleRevision,
	db.NotifyForumMention,
	db.NotifyDirectMessage,
	db.NotifyPostLike,
	db.NotifyReleaseAvailable,
	db.NotifyUpdateRolledBack,
	db.NotifyNewTicket,
	db.NotifyTicketResult,
	db.NotifyPageDeleted,
	db.NotifyAnnouncement,
}

type Notifications struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*Notifications)(nil)

func NewNotifications(d Deps, next http.Handler) *Notifications {
	return &Notifications{deps: d, next: next}
}

func (h *Notifications) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != NotificationsPath && r.URL.Path != NotificationsMutePath {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}

	if r.URL.Path == NotificationsMutePath {
		h.mute(w, r, loc, user)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.list(w, r, loc, user)
	case http.MethodDelete:
		h.clear(w, r, loc, user)
	default:
		h.next.ServeHTTP(w, r)
	}
}

func (h *Notifications) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	query := r.URL.Query()

	kinds, ok := wantedKinds(query.Get("type"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-notification-type")))
		return
	}
	var cursor *int64
	if raw := query.Get("cursor"); raw != "" && raw != "-1" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		cursor = &n
	}
	limit := defaultNotificationPage
	if n, err := strconv.Atoi(query.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	unread := query.Get("unread") == "true" || query.Get("unread") == "1"

	var found []db.Notification
	var err error
	if raw := query.Get("id"); raw != "" {
		id, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		found, err = h.deps.DB.NotificationOf(ctx, user.ID, id)
	} else {
		found, err = h.deps.DB.NotificationsOf(ctx, user.ID, cursor, unread, kinds, limit)
	}
	if err != nil {
		h.deps.log().Error("list notifications", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	current := site.FromContext(ctx)
	scheme := h.deps.Trust.Scheme(r)
	rendered := make(wikijson.Array, 0, len(found))
	ids := make([]int64, 0, len(found))
	for _, one := range found {
		ids = append(ids, one.ID)
		body, err := notificationJSON(one, current, scheme)
		if err != nil {
			h.deps.log().Error("render notification", "id", one.ID, "err", err)
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
			return
		}
		rendered = append(rendered, body)
	}

	if query.Get("mark_as_viewed") == "true" || query.Get("mark_as_viewed") == "1" {
		if err := h.deps.DB.MarkNotificationsViewed(ctx, user.ID, ids); err != nil {
			h.deps.log().Error("mark notifications", "err", err)
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
			return
		}
	}

	next := int64(-1)
	if len(ids) > 0 {
		next = ids[len(ids)-1]
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "cursor", Value: next},
		{Key: "notifications", Value: rendered},
	})
	if err != nil {
		h.deps.log().Error("encode notifications", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type clearRequest struct {
	IDs  []int64 `json:"ids"`
	All  bool    `json:"all"`
	Type string  `json:"type"`
}

func (h *Notifications) clear(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}

	raw, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var input clearRequest
	if len(raw) > 0 && json.Unmarshal(raw, &input) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-json")))
		return
	}
	if !input.All && len(input.IDs) == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}

	var removed int64
	if input.All {
		kinds, ok := wantedKinds(input.Type)
		if !ok {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-notification-type")))
			return
		}
		removed, err = h.deps.DB.DeleteAllNotifications(ctx, user.ID, kinds)
	} else {
		removed, err = h.deps.DB.DeleteNotifications(ctx, user.ID, input.IDs)
	}
	if err != nil {
		h.deps.log().Error("clear notifications", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "removed", Value: removed}})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// Direct messages have their own page, so none of the groups reach the ones
// older messages left behind.
func wantedKinds(raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "", "all":
		return kindsWhere(func(kind string) bool { return kind != db.NotifyDirectMessage }), true
	case "notices":
		return kindsWhere(func(kind string) bool {
			return kind != db.NotifyDirectMessage && !slices.Contains(db.CommentKinds, kind)
		}), true
	case "comments":
		return slices.Clone(db.CommentKinds), true
	}
	var out []string
	for _, one := range strings.Split(raw, ",") {
		one = strings.TrimSpace(one)
		if !slices.Contains(notificationKinds, one) {
			return nil, false
		}
		out = append(out, one)
	}
	return out, true
}

// The stored meta is spread over the envelope, so a reader sees the same shape
// the notification was written with plus the four fields every kind carries.
func notificationJSON(one db.Notification, current *db.Site, scheme string) (wikijson.Object, error) {
	out := wikijson.Object{
		{Key: "id", Value: one.ID},
		{Key: "type", Value: one.Type},
		{Key: "created_at", Value: isoTime(one.CreatedAt)},
		{Key: "is_viewed", Value: one.IsViewed},
	}
	if one.SiteID != 0 && (current == nil || one.SiteID != current.ID) {
		out = append(out, wikijson.Field{Key: "site", Value: wikijson.Object{
			{Key: "title", Value: one.SiteTitle},
			{Key: "url", Value: scheme + "://" + one.SiteDomain},
		}})
	}
	meta, err := decodeMeta(one.Meta)
	if err != nil {
		return nil, err
	}
	fields, ok := meta.(map[string]any)
	if !ok {
		return out, nil
	}
	for _, key := range sortedKeys(fields) {
		out = append(out, wikijson.Field{Key: key, Value: fromJSON(fields[key])})
	}
	return out, nil
}

func kindsWhere(keep func(string) bool) []string {
	var out []string
	for _, kind := range notificationKinds {
		if keep(kind) {
			out = append(out, kind)
		}
	}
	return out
}

type muteRequest struct {
	ArticleID int64 `json:"article_id"`
	ThreadID  int64 `json:"thread_id"`
	UserID    int64 `json:"user_id"`
}

func (m muteRequest) targets() int {
	n := 0
	for _, id := range []int64{m.ArticleID, m.ThreadID, m.UserID} {
		if id != 0 {
			n++
		}
	}
	return n
}

func (h *Notifications) mute(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil || (r.Method != http.MethodPost && r.Method != http.MethodDelete) {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}
	raw, err := readBody(r)
	var input muteRequest
	if err != nil || json.Unmarshal(raw, &input) != nil || input.targets() != 1 || input.UserID == user.ID {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	switch {
	case r.Method == http.MethodDelete:
		err = h.deps.DB.Unmute(ctx, user.ID, nonZero(input.ArticleID), nonZero(input.ThreadID), nonZero(input.UserID))
	case input.ArticleID != 0:
		err = h.deps.DB.MuteArticle(ctx, user.ID, input.ArticleID)
	case input.ThreadID != 0:
		err = h.deps.DB.MuteThread(ctx, user.ID, input.ThreadID)
	default:
		err = h.deps.DB.MuteUser(ctx, user.ID, input.UserID)
	}
	if err != nil {
		h.deps.log().Error("mute", "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, field("status", "ok"))
}

func nonZero(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}
