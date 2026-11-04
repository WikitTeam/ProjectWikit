package mount

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	NotifyPath      = "/pw-api/mounts/notify"
	MountHeader     = "X-Pwikit-Mount"
	SignatureHeader = "X-Pwikit-Signature"

	mostRecipients = 500
	mostTitle      = 100
	mostMessage    = 2000
	mostBody       = 64 << 10
)

type Sender interface {
	SendNotification(ctx context.Context, siteID int64, kind, meta string, recipients []int64, at time.Time) error
}

type notice struct {
	Users   []int64 `json:"users"`
	Title   string  `json:"title"`
	Message string  `json:"message"`
	Link    string  `json:"link"`
}

// NotifyHandler lets a mounted program tell readers something through the
// wiki's own notices. A program proves which mount it is with that mount's key,
// so it can only ever speak as itself.
func NotifyHandler(secrets map[string]string, sender Sender) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		from := r.Header.Get(MountHeader)
		secret, ok := secrets[from]
		if !ok {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, mostBody))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if !hmac.Equal([]byte(r.Header.Get(SignatureHeader)), []byte(Signature(secret, string(body)))) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		var n notice
		if json.Unmarshal(body, &n) != nil {
			http.Error(w, "the body is not a notice", http.StatusBadRequest)
			return
		}
		n.Title = strings.TrimSpace(n.Title)
		n.Message = strings.TrimSpace(n.Message)
		switch {
		case len(n.Users) == 0 || len(n.Users) > mostRecipients:
			http.Error(w, "users must name between 1 and 500 readers", http.StatusBadRequest)
			return
		case n.Title == "" || utf8.RuneCountInString(n.Title) > mostTitle:
			http.Error(w, "title must be 1 to 100 characters", http.StatusBadRequest)
			return
		case utf8.RuneCountInString(n.Message) > mostMessage:
			http.Error(w, "message is longer than 2000 characters", http.StatusBadRequest)
			return
		case n.Link != "" && (!strings.HasPrefix(n.Link, "/") || strings.HasPrefix(n.Link, "//") || strings.HasPrefix(n.Link, `/\`)):
			http.Error(w, "link must be a path on this site", http.StatusBadRequest)
			return
		}
		current := site.FromContext(r.Context())
		if current == nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		meta, err := json.Marshal(map[string]string{
			"title":   n.Title,
			"message": strings.ReplaceAll(html.EscapeString(n.Message), "\n", "<br>"),
			"link":    n.Link,
			"from":    from,
		})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := sender.SendNotification(r.Context(), current.ID, db.NotifyExternal, string(meta), n.Users, time.Now()); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
