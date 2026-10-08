package admin

import (
	"context"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type handled struct {
	kind    string
	id      int64
	subject string
	status  string
	reply   string
}

func worthTelling(to *int64, by int64, before, after handled, pending string) bool {
	if to == nil || *to == by || after.status == pending {
		return false
	}
	return after.status != before.status || after.reply != before.reply
}

func (h *Handler) tellSubmitter(ctx context.Context, to *int64, by int64, before, after handled, pending string) {
	if !worthTelling(to, by, before, after, pending) {
		return
	}
	meta, err := wikijson.Marshal(wikijson.Object{
		{Key: "kind", Value: after.kind},
		{Key: "ticket_id", Value: after.id},
		{Key: "subject", Value: after.subject},
		{Key: "status", Value: after.status},
		{Key: "reply", Value: after.reply},
	})
	if err == nil {
		err = h.deps.DB.SendNotification(ctx, siteID(ctx), db.NotifyTicketResult, meta, []int64{*to}, time.Now())
	}
	if err != nil {
		h.deps.logger().Error("tell the submitter", "kind", after.kind, "id", after.id, "err", err)
	}
}
