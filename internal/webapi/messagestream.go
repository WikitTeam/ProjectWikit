package webapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/inbox"
)

const MessagesStreamPath = MessagesPrefix + "stream"

// Proxies commonly drop a connection that has been silent for a minute or so.
const streamHeartbeat = 25 * time.Second

const streamRetry = 5 * time.Second

// The event only says that something changed and how many messages are unread.
// The page fetches the rest through the usual endpoints and their checks.
func (h *Messages) stream(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	changes, leave, err := h.deps.Inbox.Subscribe(user.ID)
	switch {
	case errors.Is(err, inbox.ErrFull):
		w.WriteHeader(http.StatusTooManyRequests)
		return
	case err != nil:
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer leave()

	ctx := r.Context()
	flusher := http.NewResponseController(w)
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-store")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, "retry: %d\n\n", streamRetry.Milliseconds()); err != nil {
		return
	}

	heartbeat := time.NewTicker(streamHeartbeat)
	defer heartbeat.Stop()
	for {
		unread, err := h.deps.DB.UnreadMessages(ctx, user.ID)
		if err != nil {
			if ctx.Err() == nil {
				h.deps.log().Error("messages stream", "user", user.ID, "err", err)
			}
			return
		}
		if _, err := fmt.Fprintf(w, "event: messages\ndata: {\"unread\":%d}\n\n", unread); err != nil {
			return
		}
		if flusher.Flush() != nil {
			return
		}
	wait:
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.deps.Inbox.Done():
				return
			case <-changes:
				break wait
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": \n\n"); err != nil {
					return
				}
				if flusher.Flush() != nil {
					return
				}
			}
		}
	}
}

func (h *Messages) notify(users ...int64) {
	if h.deps.Inbox != nil {
		h.deps.Inbox.Notify(users...)
	}
}
