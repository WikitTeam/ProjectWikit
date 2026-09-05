package account

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	TicketPath     = "/-/tickets/submit"
	MembershipPath = "/-/membership/password"
)

const (
	maxTicketSubject = 200
	maxTicketBody    = 20000
)

type TicketHandler struct {
	deps Deps
}

var _ http.Handler = (*TicketHandler)(nil)

func NewTickets(d Deps) *TicketHandler { return &TicketHandler{deps: d} }

func (h *TicketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	switch r.URL.Path {
	case TicketPath:
		h.submit(w, r)
	case MembershipPath:
		h.membership(w, r, current)
	default:
		notFound(w)
	}
}

func (h *TicketHandler) submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	body := strings.TrimSpace(r.PostFormValue("body"))
	if user == nil || body == "" || utf8.RuneCountInString(body) > maxTicketBody {
		back(w, r, "applied", "failed")
		return
	}
	kind := r.PostFormValue("kind")
	if kind != db.TicketKind && kind != db.MembershipApplyKind {
		kind = db.TicketKind
	}
	subject := clip(strings.TrimSpace(r.PostFormValue("subject")), maxTicketSubject)
	page := strings.TrimSpace(r.PostFormValue("page"))

	if _, err := h.deps.DB.CreateTicket(ctx, kind, subject, body, page, user.ID, time.Now()); err != nil {
		h.deps.logger().Error("submit ticket", "user", user.ID, "err", err)
		back(w, r, "applied", "failed")
		return
	}
	back(w, r, "applied", "ok")
}

func (h *TicketHandler) membership(w http.ResponseWriter, r *http.Request, current *db.Site) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil || !current.MembershipPasswordEnabled || current.MembershipPassword == "" ||
		current.MembershipPasswordRoleID == nil {
		back(w, r, "membership", "failed")
		return
	}
	given := r.PostFormValue("password")
	if subtle.ConstantTimeCompare([]byte(given), []byte(current.MembershipPassword)) != 1 {
		back(w, r, "membership", "failed")
		return
	}
	if err := h.deps.DB.GrantRole(ctx, user.ID, *current.MembershipPasswordRoleID); err != nil {
		h.deps.logger().Error("grant membership", "user", user.ID, "err", err)
		back(w, r, "membership", "failed")
		return
	}
	back(w, r, "membership", "ok")
}

func back(w http.ResponseWriter, r *http.Request, key, outcome string) {
	page := strings.Trim(strings.TrimSpace(r.PostFormValue("page")), "/")
	if page == "" {
		redirect(w, homePath, http.StatusFound)
		return
	}
	redirect(w, "/"+page+"/"+key+"/"+outcome, http.StatusFound)
}

func clip(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
