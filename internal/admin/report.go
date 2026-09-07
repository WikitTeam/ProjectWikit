package admin

import (
	"errors"
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
	found, total, err := h.deps.DB.AdminReports(r.Context(), status, perPage, (page-1)*perPage)
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
	row, err := h.deps.DB.AdminReport(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	granted, _, err := h.access(r.Context())
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.reports"), "report_form.html", map[string]any{
		"Report":   row,
		"Statuses": reportStatuses,
		"SeeAll":   granted.Has(perms.ViewReportedFullConversation),
		"CSRF":     csrf.Issue(w, r),
		"Error":    problem,
		"Action":   Prefix + reportSlug + "/" + rest,
		"Back":     Prefix + reportSlug + "/",
	})
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
	if _, err := h.deps.DB.AdminReport(ctx, id); errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	} else if err != nil {
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
	err = h.deps.DB.ReviewReport(ctx, id, status, r.PostFormValue("admin_notes"), mine.ID, time.Now())
	if err != nil {
		return err
	}
	h.noteID(r, db.AdminChanged, reportSlug, id, status)
	redirect(w, Prefix+reportSlug+"/")
	return nil
}
