package account

import (
	"crypto/subtle"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/quiz"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	TicketPath         = "/-/tickets/submit"
	MembershipPath     = "/-/membership/password"
	MembershipQuizPath = "/-/membership/quiz"
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
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
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
	case MembershipQuizPath:
		h.membershipQuiz(w, r, current)
	default:
		h.deps.notFound(w, r)
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

	now := time.Now()
	id, err := h.deps.DB.CreateTicket(ctx, siteID(ctx), kind, subject, body, page, user.ID, now)
	if err != nil {
		h.deps.logger().Error("submit ticket", "user", user.ID, "err", err)
		back(w, r, "applied", "failed")
		return
	}
	need := perms.ViewUserTickets
	if kind == db.MembershipApplyKind {
		need = perms.ReviewMembershipApplications
	}
	if err := h.tellReviewers(r, need, kind, id, subject, user, now); err != nil {
		h.deps.logger().Error("announce ticket", "ticket", id, "err", err)
	}
	back(w, r, "applied", "ok")
}

func (h *TicketHandler) tellReviewers(r *http.Request, need, kind string, id int64, subject string, author *db.User, now time.Time) error {
	ctx := r.Context()
	audience, err := repo.NewPerms(ctx, h.deps.DB).Holders(need, now)
	if err != nil {
		return err
	}
	audience = slices.DeleteFunc(audience, func(one int64) bool { return one == author.ID })
	meta, err := wikijson.Marshal(wikijson.Object{
		{Key: "kind", Value: kind},
		{Key: "ticket_id", Value: id},
		{Key: "subject", Value: subject},
		{Key: "sender_name", Value: author.DisplayLabel()},
	})
	if err != nil {
		return err
	}
	return h.deps.DB.SendNotification(ctx, siteID(ctx), db.NotifyNewTicket, meta, audience, now)
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
	if err := h.deps.DB.GrantRole(ctx, siteID(ctx), user.ID, *current.MembershipPasswordRoleID); err != nil {
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

// The quiz is graded here against the questions as they stand now, whatever
// order the reader saw them in.
func (h *TicketHandler) membershipQuiz(w http.ResponseWriter, r *http.Request, current *db.Site) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	setting, err := h.deps.DB.MembershipQuiz(ctx, current.ID)
	if err != nil {
		h.deps.logger().Error("read membership quiz", "site", current.ID, "err", err)
		back(w, r, "membership-quiz", "failed")
		return
	}
	questions := quiz.Parse(setting.Questions)
	if user == nil || !setting.Enabled || setting.RoleID == nil || len(questions) == 0 {
		back(w, r, "membership-quiz", "failed")
		return
	}
	attempts, err := h.deps.DB.QuizAttempts(ctx, current.ID, user.ID)
	if err != nil {
		h.deps.logger().Error("read quiz attempts", "user", user.ID, "err", err)
		back(w, r, "membership-quiz", "failed")
		return
	}
	now := time.Now().UTC()
	if left, _ := quiz.State(attempts, setting.Tries, quiz.Hours(setting.CooldownHours), now); left == 0 {
		back(w, r, "membership-quiz", "failed")
		return
	}

	answers := map[int][]int{}
	for i := range questions {
		for _, raw := range r.PostForm["q"+strconv.Itoa(i)] {
			if n, err := strconv.Atoi(raw); err == nil {
				answers[i] = append(answers[i], n)
			}
		}
	}
	passed := quiz.Passed(questions, answers)
	if err := h.deps.DB.AddQuizAttempt(ctx, current.ID, user.ID, now, passed); err != nil {
		h.deps.logger().Error("record quiz attempt", "user", user.ID, "err", err)
		back(w, r, "membership-quiz", "failed")
		return
	}
	if !passed {
		back(w, r, "membership-quiz", "failed")
		return
	}
	if err := h.deps.DB.GrantRole(ctx, current.ID, user.ID, *setting.RoleID); err != nil {
		h.deps.logger().Error("grant membership", "user", user.ID, "err", err)
		back(w, r, "membership-quiz", "failed")
		return
	}
	back(w, r, "membership-quiz", "ok")
}
