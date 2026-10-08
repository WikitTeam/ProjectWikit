package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const reportSlug = "reports"

var reportStatuses = []string{db.ReportPending, db.ReportReviewed, db.ReportDismissed}

func init() {
	register(screen{slug: reportSlug, label: "admin.reports", need: perms.ViewUserReports, serve: (*Handler).reports})
}

func (h *Handler) reports(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+reportSlug), "/")
	if r.Method == http.MethodPost {
		return h.saveReport(w, r, loc, rest)
	}
	if rest == "" {
		return h.reportList(w, r, loc)
	}
	return h.reportForm(w, r, loc, rest, "")
}

func (h *Handler) reportList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	status := r.URL.Query().Get("status")
	if !contains(reportStatuses, status) {
		status = ""
	}
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	found, total, err := h.deps.DB.AdminReports(r.Context(), siteID(r.Context()), status, perPage, (page-1)*perPage)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.reports"), "report_list.html", map[string]any{
		"Reports":  found,
		"Status":   status,
		"Statuses": reportStatuses,
		"Page":     page,
		"Pages":    (total + perPage - 1) / perPage,
		"Total":    total,
		"Base":     Prefix + reportSlug + "/",
		"Action":   Prefix + reportSlug + "/",
	})
}

func (h *Handler) reportForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.AdminReport(r.Context(), siteID(r.Context()), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	conversation := ""
	if mine := auth.FromContext(r.Context()); mine != nil && mine.IsSuperuser {
		if conversation, err = h.fullConversation(r.Context(), loc, id); err != nil {
			return err
		}
	}
	return h.page(w, r, loc, loc.T("admin.reports"), "report_form.html", map[string]any{
		"Report":       row,
		"Messages":     reportedMessages(row.Messages, site.Zone(r.Context())),
		"Statuses":     reportStatuses,
		"Conversation": conversation,
		"CSRF":         csrf.Issue(w, r),
		"Error":        problem,
		"Action":       Prefix + reportSlug + "/" + rest,
		"Back":         Prefix + reportSlug + "/",
	})
}

func reportedMessages(raw string, zone *time.Location) string {
	var found []struct {
		SenderName string    `json:"sender_name"`
		Body       string    `json:"body"`
		CreatedAt  time.Time `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(raw), &found); err != nil {
		return raw
	}
	var b strings.Builder
	for _, one := range found {
		fmt.Fprintf(&b, "%s %s: %s\n", one.CreatedAt.In(zone).Format("2006-01-02 15:04"), one.SenderName, one.Body)
	}
	return b.String()
}

func (h *Handler) fullConversation(ctx context.Context, loc *i18n.Localizer, id int64) (string, error) {
	report, err := h.deps.DB.Report(ctx, siteID(ctx), id)
	if err != nil {
		return "", err
	}
	if report.ReporterID == nil || report.ReportedID == nil {
		return "", nil
	}
	names := map[int64]string{}
	for _, uid := range []int64{*report.ReporterID, *report.ReportedID} {
		user, err := h.deps.DB.UserByID(ctx, uid)
		if errors.Is(err, db.ErrNotFound) {
			continue
		}
		if err != nil {
			return "", err
		}
		names[uid] = user.DisplayLabel()
	}
	found, err := h.deps.DB.MessagesBetween(ctx, *report.ReporterID, *report.ReportedID)
	if err != nil {
		return "", err
	}
	zone := site.Zone(ctx)
	var b strings.Builder
	for _, one := range found {
		name, ok := names[one.SenderID]
		if !ok {
			name = loc.T("user-deleted")
		}
		fmt.Fprintf(&b, "%s %s: %s\n", one.CreatedAt.In(zone).Format("2006-01-02 15:04"), name, one.Body)
	}
	return b.String(), nil
}

func (h *Handler) saveReport(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	stored, err := h.deps.DB.AdminReport(ctx, siteID(ctx), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}

	status := r.PostFormValue("status")
	if !contains(reportStatuses, status) {
		return h.reportForm(w, r, loc, rest, loc.T("admin.report-bad-status"))
	}
	mine := auth.FromContext(ctx)
	if mine == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	reply := strings.TrimSpace(r.PostFormValue("reply"))
	err = h.deps.DB.ReviewReport(ctx, siteID(ctx), id, status, r.PostFormValue("admin_notes"), reply, mine.ID, time.Now())
	if err != nil {
		return err
	}
	h.tellSubmitter(ctx, stored.ReporterID, mine.ID,
		handled{kind: db.ReportKind, id: id, subject: stored.Reported, status: stored.Status, reply: stored.Reply},
		handled{kind: db.ReportKind, id: id, subject: stored.Reported, status: status, reply: reply}, db.ReportPending)
	h.noteID(r, db.AdminChanged, reportSlug, id, status)
	redirect(w, Prefix+reportSlug+"/")
	return nil
}
