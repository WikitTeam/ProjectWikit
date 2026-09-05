package webapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	maxReportReason      = 2000
	maxReportedMessages  = 100
	maxReportsPerDay     = 5
	reportRateLimitScope = 24 * time.Hour
)

type reportRequest struct {
	ReportedID int64   `json:"reported_id"`
	MessageIDs []int64 `json:"message_ids"`
	Reason     string  `json:"reason"`
}

func (h *Messages) report(r *http.Request, loc *i18n.Localizer, user *db.User) (string, int, error) {
	ctx := r.Context()
	raw, err := readBody(r)
	if err != nil {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-bad-request"))
	}
	var input reportRequest
	if json.Unmarshal(raw, &input) != nil {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-bad-json"))
	}
	reason := strings.TrimSpace(input.Reason)
	switch {
	case input.ReportedID == 0:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-missing-reported"))
	case len(input.MessageIDs) == 0:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-no-messages-picked"))
	case len(input.MessageIDs) > maxReportedMessages:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-too-many-messages", "max", maxReportedMessages))
	case reason == "":
		return "", 0, deny(http.StatusBadRequest, loc.T("api-missing-reason"))
	case utf8.RuneCountInString(reason) > maxReportReason:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-reason-too-long", "max", maxReportReason))
	case input.ReportedID == user.ID:
		return "", 0, deny(http.StatusBadRequest, loc.T("api-cannot-report-self"))
	}

	reported, err := h.deps.DB.UserByID(ctx, input.ReportedID)
	if errors.Is(err, db.ErrNotFound) {
		return "", 0, errNotFound
	}
	if err != nil {
		return "", 0, err
	}

	recent, err := h.deps.DB.ReportsSince(ctx, user.ID, reported.ID, time.Now().Add(-reportRateLimitScope))
	if err != nil {
		return "", 0, err
	}
	if recent >= maxReportsPerDay {
		return "", 0, deny(http.StatusTooManyRequests, loc.T("api-report-rate-limited"))
	}

	found, err := h.deps.DB.MessagesByIDs(ctx, user.ID, reported.ID, input.MessageIDs)
	if err != nil {
		return "", 0, err
	}
	if len(found) != len(distinct(input.MessageIDs)) {
		return "", 0, deny(http.StatusBadRequest, loc.T("api-messages-not-in-conversation"))
	}

	snapshot, err := h.snapshot(r, loc, found)
	if err != nil {
		return "", 0, err
	}
	id, err := h.deps.DB.CreateReport(ctx, user.ID, reported.ID, reason, snapshot, time.Now())
	if err != nil {
		return "", 0, err
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "status", Value: "ok"},
		{Key: "report_id", Value: id},
	})
	return body, http.StatusOK, err
}

func (h *Messages) snapshot(r *http.Request, loc *i18n.Localizer, found []db.DirectMessage) (string, error) {
	out := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		record, err := h.messageRecord(r, loc, one)
		if err != nil {
			return "", err
		}
		out = append(out, record)
	}
	return wikijson.Marshal(out)
}

func (h *Messages) messageRecord(r *http.Request, loc *i18n.Localizer, m db.DirectMessage) (wikijson.Object, error) {
	sender, err := h.deps.DB.UserByID(r.Context(), m.SenderID)
	name := loc.T("user-deleted")
	if err == nil {
		name = sender.DisplayLabel()
	} else if !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	return wikijson.Object{
		{Key: "id", Value: m.ID},
		{Key: "sender_id", Value: m.SenderID},
		{Key: "sender_name", Value: name},
		{Key: "body", Value: m.Body},
		{Key: "created_at", Value: isoTime(m.CreatedAt)},
	}, nil
}

func distinct(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
