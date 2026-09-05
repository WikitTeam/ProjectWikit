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

const NotificationsPath = "/pw-api/notifications"

const defaultNotificationPage = 10

// The reader may ask for one kind at a time. Anything the catalogue does not
// know is refused rather than quietly widened to everything.
var notificationKinds = []string{
	db.NotifyWelcome,
	db.NotifyNewPostReply,
	db.NotifyNewThreadPost,
	db.NotifyNewArticleRevision,
	db.NotifyForumMention,
	db.NotifyDirectMessage,
	db.NotifyPostLike,
}

type Notifications struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Notifications)(nil)

func NewNotifications(d Deps, upstream http.Handler) *Notifications {
	return &Notifications{deps: d, upstream: upstream}
}

func (h *Notifications) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != NotificationsPath {
		h.upstream.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.list(w, r, loc, user)
	case http.MethodDelete:
		h.clear(w, r, loc, user)
	default:
		h.upstream.ServeHTTP(w, r)
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

	found, err := h.deps.DB.NotificationsOf(ctx, user.ID, cursor, unread, kinds, limit)
	if err != nil {
		h.deps.log().Error("list notifications", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	ids := make([]int64, 0, len(found))
	for _, one := range found {
		ids = append(ids, one.ID)
		body, err := notificationJSON(one)
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
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-csrf-failed")))
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

func wantedKinds(raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return nil, true
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
func notificationJSON(one db.Notification) (wikijson.Object, error) {
	out := wikijson.Object{
		{Key: "id", Value: one.ID},
		{Key: "type", Value: one.Type},
		{Key: "created_at", Value: isoTime(one.CreatedAt)},
		{Key: "is_viewed", Value: one.IsViewed},
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
