package admin

import (
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/update"
)

const updateSlug = "update"

const (
	checkParam  = "update-check"
	checkNewest = "newest"
	checkFound  = "found"
	checkFailed = "failed"
)

type updateNotice struct {
	Error    bool
	Text     string
	Detail   string
	Notes    string
	Actions  bool
	StartNow bool
	CSRF     string
}

func (h *Handler) updateNotices(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) []updateNotice {
	if h.deps.Updates == nil {
		return nil
	}
	user := auth.FromContext(r.Context())
	super := user != nil && user.IsSuperuser
	var out []updateNotice
	if h.deps.Updates.AsRoot && super {
		out = append(out, updateNotice{Error: true, Text: loc.T("update.notice-as-root")})
	}
	if exposed := h.deps.Updates.Exposed; exposed != "" && super {
		out = append(out, updateNotice{Error: true, Text: loc.T("update.notice-exposed"), Detail: exposed})
	}
	if problem, ok := h.deps.Updates.Problem(r.Context()); ok && super {
		out = append(out, updateNotice{
			Error: true,
			Text: loc.T("update.notice-check-"+problem.Kind, "log", escape.HTML(h.deps.Updates.LogPath),
				"version", escape.HTML(problem.Version)),
			Detail: problem.Error,
		})
	}
	for _, n := range h.deps.Updates.Notices(r.Context()) {
		v := updateNotice{Notes: n.Notes}
		switch n.Kind {
		case update.NoticeScheduled:
			v.Text = loc.T("update.notice-scheduled", "version", escape.HTML(n.Version), "time", localTime(n.At))
			v.Actions = super
			v.StartNow = super && !n.ByHand && n.At.After(time.Now().Add(update.AnnounceAhead)) && h.deps.Updates.Startable(r.Context())
		case update.NoticeAvailable:
			v.Text = loc.T("update.notice-available", "version", escape.HTML(n.Version))
			v.Detail = loc.T("update.reason-"+n.Reason.Code, "detail", n.Reason.Detail)
			v.StartNow = super && h.deps.Updates.Startable(r.Context())
		case update.NoticeUpdated:
			v.Text = loc.T("update.notice-updated", "version", escape.HTML(n.Version), "from", escape.HTML(n.From))
		case update.NoticeRolledBack:
			v.Error = true
			v.Text = loc.T("update.notice-rolled-back", "version", escape.HTML(n.Version), "from", escape.HTML(n.From), "time", localTime(n.At))
			if super {
				v.Detail = n.Error
			}
		}
		if (v.Actions || v.StartNow) && w != nil {
			v.CSRF = csrf.Issue(w, r)
		}
		out = append(out, v)
	}
	return out
}

func localTime(at time.Time) string {
	utc := at.UTC()
	return `<time datetime="` + utc.Format(time.RFC3339) + `" data-local-time>` + utc.Format("2006-01-02 15:04") + ` UTC</time>`
}

func localDateTime(at time.Time) string {
	utc := at.UTC()
	return `<time datetime="` + utc.Format(time.RFC3339) + `" data-local-time="split">` + utc.Format("2006-01-02") +
		`<br><strong>` + utc.Format("15:04:05") + ` UTC</strong></time>`
}

func timeOf(at any) (time.Time, bool) {
	switch v := at.(type) {
	case time.Time:
		return v, !v.IsZero()
	case *time.Time:
		if v == nil {
			return time.Time{}, false
		}
		return *v, !v.IsZero()
	}
	return time.Time{}, false
}

func (h *Handler) updateAction(w http.ResponseWriter, r *http.Request, action string) error {
	user := auth.FromContext(r.Context())
	if h.deps.Updates == nil || user == nil || !user.IsSuperuser || r.Method != http.MethodPost {
		h.next.ServeHTTP(w, r)
		return nil
	}
	if !h.verified(w, r) {
		return nil
	}
	var err error
	switch action {
	case "postpone":
		err = h.deps.Updates.Postpone(r.Context())
	case "skip":
		err = h.deps.Updates.Skip(r.Context())
	case "now":
		err = h.deps.Updates.StartNow(r.Context())
	case "check":
		return h.checkForUpdate(w, r)
	default:
		h.next.ServeHTTP(w, r)
		return nil
	}
	if err != nil {
		return err
	}
	h.note(r, db.AdminChanged, updateSlug, action, "")
	seeOther(w, Prefix, http.StatusSeeOther)
	return nil
}

type checkResult struct {
	Error  bool
	Text   string
	Detail string
}

func (h *Handler) checkResult(r *http.Request, loc *i18n.Localizer) *checkResult {
	switch r.URL.Query().Get(checkParam) {
	case checkNewest:
		return &checkResult{Text: loc.T("update.check-newest", "version", h.deps.Updates.Current)}
	case checkFailed:
		return &checkResult{Error: true, Text: loc.T("update.check-failed"), Detail: h.deps.Updates.State(r.Context()).CheckError}
	}
	return nil
}

func (h *Handler) checkForUpdate(w http.ResponseWriter, r *http.Request) error {
	st, err := h.deps.Updates.CheckNow(r.Context())
	if err != nil {
		return err
	}
	h.note(r, db.AdminChanged, updateSlug, "check", "")
	result := checkNewest
	switch {
	case st.CheckError != "":
		result = checkFailed
	case update.Newer(st.LatestVersion, h.deps.Updates.Current):
		result = checkFound
	}
	seeOther(w, Prefix+"?"+checkParam+"="+result, http.StatusSeeOther)
	return nil
}
