package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const MessagesPrefix = "/pw-api/messages/"

const (
	maxMessageLength   = 4000
	messagePreviewSize = 150
	defaultMessagePage = 30
)

type Messages struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Messages)(nil)

func NewMessages(d Deps, upstream http.Handler) *Messages {
	return &Messages{deps: d, upstream: upstream}
}

func (h *Messages) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, MessagesPrefix)
	if !ok {
		h.upstream.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	user := auth.FromContext(r.Context())

	head, tail, _ := strings.Cut(rest, "/")
	switch {
	case head == "conversations" && tail == "" && r.Method == http.MethodGet:
		h.guard(w, r, loc, user, false, func() (string, int, error) { return h.conversations(r, user) })
	case head == "with" && r.Method == http.MethodGet:
		h.guard(w, r, loc, user, false, func() (string, int, error) { return h.conversation(r, user, tail) })
	case head == "can-send" && r.Method == http.MethodGet:
		h.guard(w, r, loc, user, false, func() (string, int, error) { return h.canSend(r, loc, user, tail) })
	case head == "send" && tail == "" && r.Method == http.MethodPost:
		h.guard(w, r, loc, user, true, func() (string, int, error) { return h.send(r, loc, user) })
	case head == "report" && tail == "" && r.Method == http.MethodPost:
		h.guard(w, r, loc, user, true, func() (string, int, error) { return h.report(r, loc, user) })
	default:
		h.upstream.ServeHTTP(w, r)
	}
}

func (h *Messages) guard(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User, writes bool, answer func() (string, int, error)) {
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	if writes {
		current := site.FromContext(r.Context())
		if current == nil {
			h.deps.log().Error("messages api", "err", errors.New("the request carries no site"))
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
			return
		}
		if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
			writeJSON(w, http.StatusForbidden, field("error", loc.T("api-csrf-failed")))
			return
		}
	}
	body, status, err := answer()
	if err != nil {
		h.fail(w, loc, r, user, err)
		return
	}
	writeJSON(w, status, body)
}

func (h *Messages) fail(w http.ResponseWriter, loc *i18n.Localizer, r *http.Request, user *db.User, err error) {
	var denied *denial
	switch {
	case errors.As(err, &denied):
		writeJSON(w, denied.status, field("error", denied.message))
	case errors.Is(err, errNotFound):
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-user-not-found")))
	default:
		h.deps.log().Error("messages api", "path", r.URL.Path, "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
	}
}

type denial struct {
	status  int
	message string
}

func (d *denial) Error() string { return d.message }

func deny(status int, message string) error { return &denial{status: status, message: message} }

func (h *Messages) conversations(r *http.Request, user *db.User) (string, int, error) {
	ctx := r.Context()
	found, err := h.deps.DB.Conversations(ctx, user.ID)
	if err != nil {
		return "", 0, err
	}
	out := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		partner, err := h.userJSON(ctx, one.PartnerID)
		if err != nil {
			return "", 0, err
		}
		out = append(out, wikijson.Object{
			{Key: "partner", Value: partner},
			{Key: "last_message", Value: wikijson.Object{
				{Key: "id", Value: one.Last.ID},
				{Key: "sender_id", Value: one.Last.SenderID},
				{Key: "preview", Value: preview(one.Last.Body)},
				{Key: "created_at", Value: isoTime(one.Last.CreatedAt)},
			}},
			{Key: "unread_count", Value: one.Unread},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "conversations", Value: out}})
	return body, http.StatusOK, err
}

func (h *Messages) conversation(r *http.Request, user *db.User, raw string) (string, int, error) {
	ctx := r.Context()
	partner, err := h.partner(ctx, raw)
	if err != nil {
		return "", 0, err
	}
	query := r.URL.Query()
	limit := defaultMessagePage
	if n, err := strconv.Atoi(query.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}

	var found []db.DirectMessage
	next := int64(-1)
	if after, err := strconv.ParseInt(query.Get("after"), 10, 64); err == nil && after >= 0 {
		found, err = h.deps.DB.ConversationAfter(ctx, user.ID, partner.ID, after, limit)
		if err != nil {
			return "", 0, err
		}
	} else {
		var before *int64
		if cursor, err := strconv.ParseInt(query.Get("cursor"), 10, 64); err == nil && cursor >= 0 {
			before = &cursor
		}
		found, err = h.deps.DB.ConversationBefore(ctx, user.ID, partner.ID, before, limit)
		if err != nil {
			return "", 0, err
		}
		if len(found) > 0 {
			next = found[len(found)-1].ID
		}
	}

	if query.Get("mark_read") == "true" || query.Get("mark_read") == "1" {
		if _, err := h.deps.DB.MarkConversationRead(ctx, user.ID, partner.ID); err != nil {
			return "", 0, err
		}
	}

	partnerJSON, err := repo.UserJSON(ctx, h.deps.DB, partner)
	if err != nil {
		return "", 0, err
	}
	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		rendered = append(rendered, messageJSON(one))
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "partner", Value: partnerJSON},
		{Key: "messages", Value: rendered},
		{Key: "cursor", Value: next},
	})
	return body, http.StatusOK, err
}

func (h *Messages) canSend(r *http.Request, loc *i18n.Localizer, user *db.User, raw string) (string, int, error) {
	ctx := r.Context()
	recipient, err := h.partner(ctx, raw)
	if err != nil {
		return "", 0, err
	}
	var reason any
	allowed := true
	if err := h.mayMessage(ctx, loc, user, recipient); err != nil {
		var denied *denial
		if !errors.As(err, &denied) {
			return "", 0, err
		}
		allowed, reason = false, denied.message
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "allowed", Value: allowed},
		{Key: "reason", Value: reason},
	})
	return body, http.StatusOK, err
}

func (h *Messages) partner(ctx context.Context, raw string) (*db.User, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, errNotFound
	}
	found, err := h.deps.DB.UserByID(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errNotFound
	}
	return found, err
}

func (h *Messages) userJSON(ctx context.Context, id int64) (wikijson.Object, error) {
	found, err := h.deps.DB.UserByID(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return repo.UserJSON(ctx, h.deps.DB, nil)
	}
	if err != nil {
		return nil, err
	}
	return repo.UserJSON(ctx, h.deps.DB, found)
}

func messageJSON(m db.DirectMessage) wikijson.Object {
	return wikijson.Object{
		{Key: "id", Value: m.ID},
		{Key: "sender_id", Value: m.SenderID},
		{Key: "recipient_id", Value: m.RecipientID},
		{Key: "body", Value: m.Body},
		{Key: "created_at", Value: isoTime(m.CreatedAt)},
		{Key: "is_read", Value: m.IsRead},
	}
}

func preview(body string) string {
	if utf8.RuneCountInString(body) <= messagePreviewSize {
		return body
	}
	return string([]rune(body)[:messagePreviewSize]) + "…"
}

type sendRequest struct {
	RecipientID int64  `json:"recipient_id"`
	Body        string `json:"body"`
}

func (h *Messages) send(r *http.Request, loc *i18n.Localizer, user *db.User) (string, int, error) {
	ctx := r.Context()
	raw, err := readBody(r)
	if err != nil {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-bad-request"))
	}
	var input sendRequest
	if json.Unmarshal(raw, &input) != nil {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-bad-json"))
	}
	if input.RecipientID == 0 {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-missing-recipient"))
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-empty-message"))
	}
	if utf8.RuneCountInString(body) > maxMessageLength {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-message-too-long", "max", maxMessageLength))
	}

	recipient, err := h.deps.DB.UserByID(ctx, input.RecipientID)
	if errors.Is(err, db.ErrNotFound) {
		return "", 0, errNotFound
	}
	if err != nil {
		return "", 0, err
	}
	if err := h.mayMessage(ctx, loc, user, recipient); err != nil {
		return "", 0, err
	}

	sent, err := h.deps.DB.SendDirectMessage(ctx, user.ID, recipient.ID, body, time.Now())
	if err != nil {
		return "", 0, err
	}
	meta, err := wikijson.Marshal(wikijson.Object{
		{Key: "sender_id", Value: user.ID},
		{Key: "sender_name", Value: user.DisplayLabel()},
		{Key: "message_id", Value: sent.ID},
		{Key: "preview", Value: preview(sent.Body)},
	})
	if err != nil {
		return "", 0, err
	}
	if err := h.deps.DB.SendNotification(ctx, db.NotifyDirectMessage, meta, []int64{recipient.ID}, sent.CreatedAt); err != nil {
		return "", 0, err
	}
	out, err := wikijson.Marshal(messageJSON(sent))
	return out, http.StatusOK, err
}

func (h *Messages) mayMessage(ctx context.Context, loc *i18n.Localizer, sender, recipient *db.User) error {
	if sender.ID == recipient.ID {
		return deny(http.StatusForbidden, loc.T("api-cannot-message-self"))
	}
	if !recipient.IsActive {
		return deny(http.StatusForbidden, loc.T("api-recipient-inactive"))
	}
	if !sender.CanSendDirectMessages {
		return deny(http.StatusForbidden, loc.T("api-messaging-disabled"))
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(sender, time.Now())
	if err != nil {
		return err
	}
	if !perms.Resolve(subject, nil).Has(perms.SendDirectMessage) {
		return deny(http.StatusForbidden, loc.T("api-no-message-permission"))
	}
	blocked, err := h.deps.DB.IsBlocked(ctx, recipient.ID, sender.ID)
	if err != nil {
		return err
	}
	if blocked {
		return deny(http.StatusForbidden, loc.T("api-blocked-by-recipient"))
	}
	return nil
}
