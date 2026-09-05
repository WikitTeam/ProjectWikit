package webapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const AdminPrefix = "/pw-api/admin/"

type Admin struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Admin)(nil)

func NewAdmin(d Deps, upstream http.Handler) *Admin {
	return &Admin{deps: d, upstream: upstream}
}

func (h *Admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, AdminPrefix)
	if !ok || r.Method != http.MethodGet {
		h.upstream.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	head, tail, _ := strings.Cut(rest, "/")
	switch {
	case head == "sus" && tail == "":
		h.suspicious(w, r, loc)
	case head == "reports" && strings.HasSuffix(tail, "/full-conversation"):
		h.conversation(w, r, loc, strings.TrimSuffix(tail, "/full-conversation"))
	default:
		h.upstream.ServeHTTP(w, r)
	}
}

func (h *Admin) suspicious(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	if !h.allowed(w, r, loc, perms.ViewSensitiveInfo) {
		return
	}
	found, err := h.deps.DB.SuspiciousUsers(ctx)
	if err != nil {
		h.deps.log().Error("list suspicious users", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	out := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		var ip any
		if one.IP != nil {
			ip = *one.IP
		}
		out = append(out, wikijson.Object{
			{Key: "user", Value: wikijson.Object{
				{Key: "id", Value: one.UserID},
				{Key: "name", Value: one.Username},
			}},
			{Key: "ip", Value: ip},
		})
	}
	body, err := wikijson.Marshal(out)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Admin) conversation(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, raw string) {
	ctx := r.Context()
	if !h.allowed(w, r, loc, perms.ViewUserReports, perms.ViewReportedFullConversation) {
		return
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-report-not-found")))
		return
	}
	report, err := h.deps.DB.Report(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-report-not-found")))
		return
	}
	if err != nil {
		h.deps.log().Error("read report", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if report.ReporterID == nil || report.ReportedID == nil {
		writeJSON(w, http.StatusGone, field("error", loc.T("api-report-party-gone")))
		return
	}

	reporter, err := h.deps.DB.UserByID(ctx, *report.ReporterID)
	if err != nil {
		h.reportFailed(w, loc, err)
		return
	}
	reported, err := h.deps.DB.UserByID(ctx, *report.ReportedID)
	if err != nil {
		h.reportFailed(w, loc, err)
		return
	}
	found, err := h.deps.DB.MessagesBetween(ctx, reporter.ID, reported.ID)
	if err != nil {
		h.reportFailed(w, loc, err)
		return
	}

	names := map[int64]string{reporter.ID: reporter.DisplayLabel(), reported.ID: reported.DisplayLabel()}
	messages := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		name, ok := names[one.SenderID]
		if !ok {
			name = loc.T("user-deleted")
		}
		messages = append(messages, wikijson.Object{
			{Key: "id", Value: one.ID},
			{Key: "sender_id", Value: one.SenderID},
			{Key: "sender_name", Value: name},
			{Key: "body", Value: one.Body},
			{Key: "created_at", Value: isoTime(one.CreatedAt)},
		})
	}

	reporterJSON, err := repo.UserJSON(ctx, h.deps.DB, reporter)
	if err != nil {
		h.reportFailed(w, loc, err)
		return
	}
	reportedJSON, err := repo.UserJSON(ctx, h.deps.DB, reported)
	if err != nil {
		h.reportFailed(w, loc, err)
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "reporter", Value: reporterJSON},
		{Key: "reported", Value: reportedJSON},
		{Key: "messages", Value: messages},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Admin) reportFailed(w http.ResponseWriter, loc *i18n.Localizer, err error) {
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusGone, field("error", loc.T("api-report-party-gone")))
		return
	}
	h.deps.log().Error("read reported conversation", "err", err)
	writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
}

func (h *Admin) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, wanted ...string) bool {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return false
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(user, time.Now())
	if err != nil {
		h.deps.log().Error("resolve permissions", "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return false
	}
	held := perms.Resolve(subject, nil)
	for _, name := range wanted {
		if !held.Has(name) {
			writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
			return false
		}
	}
	return true
}
