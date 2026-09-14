package update

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

const (
	AnnounceAhead = 30 * time.Minute
	DefaultWindow = "03:00-05:00"
	DefaultMinAge = 24 * time.Hour

	MissedAfter = 2 * time.Hour

	RollbackKept = 7 * 24 * time.Hour
	PostponeFor  = 24 * time.Hour
	UpdatedShown = 24 * time.Hour
)

type Settings struct {
	Auto         bool
	PublicBanner bool
	Check        bool
	Window       Window
	MinAge       time.Duration
	Mirror       string
}

type Window struct {
	start time.Duration
	end   time.Duration
}

func ParseWindow(s string) (Window, error) {
	from, to, ok := strings.Cut(strings.ReplaceAll(s, " ", ""), "-")
	if !ok {
		return Window{}, fmt.Errorf("update window %q is not two times such as 03:00-05:00", s)
	}
	start, err := clock(from)
	if err != nil {
		return Window{}, fmt.Errorf("update window %q: %w", s, err)
	}
	end, err := clock(to)
	if err != nil {
		return Window{}, fmt.Errorf("update window %q: %w", s, err)
	}
	w := Window{start: start, end: end}
	if w.length() < AnnounceAhead+time.Minute {
		return Window{}, fmt.Errorf("update window %q is shorter than the %s an update is announced ahead", s, AnnounceAhead)
	}
	return w, nil
}

func clock(s string) (time.Duration, error) {
	h, m, ok := strings.Cut(s, ":")
	hours, herr := strconv.Atoi(h)
	minutes, merr := strconv.Atoi(m)
	if !ok || herr != nil || merr != nil || hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return 0, fmt.Errorf("%q is not a time such as 03:00", s)
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}

func (w Window) length() time.Duration {
	if w.end > w.start {
		return w.end - w.start
	}
	return 24*time.Hour - w.start + w.end
}

func (w Window) String() string {
	format := func(d time.Duration) string {
		return fmt.Sprintf("%02d:%02d", int(d.Hours()), int(d.Minutes())%60)
	}
	return format(w.start) + "-" + format(w.end)
}

func (w Window) NextCheck(now time.Time, r *rand.Rand) time.Time {
	span := w.length() - AnnounceAhead
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	base := midnight.Add(w.start)
	if !base.Add(span).After(now) {
		midnight = midnight.AddDate(0, 0, 1)
		base = midnight.Add(w.start)
	}
	at := base.Add(time.Duration(r.Int64N(int64(span))))
	if !at.After(now) {
		at = now.Add(time.Minute)
	}
	return at
}

type Facts struct {
	Current string

	BundledPostgres string

	Container bool
	Now       time.Time
}

type Reason struct {
	Code   string
	Detail string
}

var reasonText = map[string]string{
	"check-off": "checking for releases is off",
	"container": "pwikit runs in a container, where a new image replaces it",
	"unknown":   "no release is known yet",
	"no-number": "this build carries no release number",
	"newest":    "this is the newest release",
	"pinned":    "held at %s until pwikit update unpin",
	"auto-off":  "automatic updates are off",
	"skipped":   "%s was skipped",
	"failed":    "%s failed before and was rolled back",
	"postponed": "postponed until %s",
	"too-new":   "released less than %s hours ago",
	"postgres":  "it moves the bundled PostgreSQL to %s, which pwikit update does by hand",
}

func (r Reason) String() string {
	text := reasonText[r.Code]
	if strings.Contains(text, "%s") {
		return fmt.Sprintf(text, r.Detail)
	}
	return text
}

func Eligible(st db.UpdateState, s Settings, f Facts) (string, Reason) {
	latest := st.LatestVersion
	no := func(code, detail string) (string, Reason) { return "", Reason{Code: code, Detail: detail} }
	switch {
	case !s.Check:
		return no("check-off", "")
	case f.Container:
		return no("container", "")
	case latest == "":
		return no("unknown", "")
	case !isRelease(f.Current):
		return no("no-number", "")
	case !Newer(latest, f.Current):
		return no("newest", "")
	case st.PinnedVersion != "":
		return no("pinned", st.PinnedVersion)
	case !s.Auto:
		return no("auto-off", "")
	case st.SkippedVersion == latest:
		return no("skipped", latest)
	case slices.Contains(st.FailedVersions, latest):
		return no("failed", latest)
	case st.PostponedUntil != nil && f.Now.Before(*st.PostponedUntil):
		return no("postponed", st.PostponedUntil.Format(time.RFC3339))
	case st.LatestPublishedAt != nil && f.Now.Sub(*st.LatestPublishedAt) < s.MinAge:
		return no("too-new", strconv.FormatFloat(s.MinAge.Hours(), 'f', -1, 64))
	case f.BundledPostgres != "" && st.LatestPostgres != "" &&
		PostgresMajor(st.LatestPostgres) != PostgresMajor(f.BundledPostgres):
		return no("postgres", PostgresMajor(st.LatestPostgres))
	}
	return latest, Reason{}
}

func isRelease(v string) bool {
	_, ok := ParseVersion(v)
	return ok
}

func Tick(st *db.UpdateState, s Settings, f Facts, fetch func() (Manifest, error), r *rand.Rand) string {
	now := f.Now
	if !s.Check {
		clearSchedule(st)
		return ""
	}
	if st.NextCheckAt == nil {
		next := s.Window.NextCheck(now, r)
		st.NextCheckAt = &next
		return ""
	}
	if !now.Before(*st.NextCheckAt) {
		checked := now
		st.CheckedAt = &checked
		if m, err := fetch(); err != nil {
			st.CheckError = err.Error()
		} else {
			st.CheckError = ""
			st.LatestVersion = m.Version
			st.LatestPostgres = m.Postgres
			st.LatestNotes = m.Notes
			if published := m.Published(); !published.IsZero() {
				st.LatestPublishedAt = &published
			} else {
				st.LatestPublishedAt = nil
			}
		}
		next := s.Window.NextCheck(now.Add(time.Minute), r)
		st.NextCheckAt = &next
		if version, _ := Eligible(*st, s, f); version != "" {
			if st.ScheduledVersion == "" || st.ScheduledAt == nil {
				at := now.Add(AnnounceAhead)
				st.ScheduledAt = &at
			}
			st.ScheduledVersion = version
		}
	}

	if st.ScheduledVersion == "" || st.ScheduledAt == nil {
		return ""
	}
	version, _ := Eligible(*st, s, f)
	if version == "" || now.After(st.ScheduledAt.Add(MissedAfter)) {
		clearSchedule(st)
		return ""
	}
	st.ScheduledVersion = version
	if now.Before(*st.ScheduledAt) {
		return ""
	}
	return version
}

func clearSchedule(st *db.UpdateState) {
	st.ScheduledVersion = ""
	st.ScheduledAt = nil
}

func Postpone(st *db.UpdateState, now time.Time) {
	until := now.Add(PostponeFor)
	st.PostponedUntil = &until
	clearSchedule(st)
}

func Skip(st *db.UpdateState) string {
	version := st.ScheduledVersion
	if version == "" {
		version = st.LatestVersion
	}
	st.SkippedVersion = version
	clearSchedule(st)
	return version
}

func Banner(st db.UpdateState, s Settings, now time.Time) (time.Time, bool) {
	if !s.Auto || !s.PublicBanner || st.ScheduledVersion == "" || st.ScheduledAt == nil {
		return time.Time{}, false
	}
	at := *st.ScheduledAt
	if now.Before(at.Add(-AnnounceAhead)) || now.After(at.Add(MissedAfter)) {
		return time.Time{}, false
	}
	return at, true
}

type NoticeKind string

const (
	NoticeScheduled  NoticeKind = "scheduled"
	NoticeAvailable  NoticeKind = "available"
	NoticeUpdated    NoticeKind = "updated"
	NoticeRolledBack NoticeKind = "rolled-back"
)

type Notice struct {
	Kind    NoticeKind
	Version string
	From    string
	At      time.Time
	Reason  Reason
	Error   string
	Notes   string
}

func Notices(st db.UpdateState, s Settings, f Facts) []Notice {
	var out []Notice
	if st.LastOutcome == OutcomeRolledBack && st.LastAt != nil && f.Now.Sub(*st.LastAt) < RollbackKept {
		out = append(out, Notice{Kind: NoticeRolledBack, Version: st.LastTo, From: st.LastFrom, At: *st.LastAt, Error: st.LastError})
	}
	if st.LastOutcome == OutcomeUpdated && st.LastAt != nil && f.Now.Sub(*st.LastAt) < UpdatedShown && st.LastTo == f.Current {
		out = append(out, Notice{Kind: NoticeUpdated, Version: st.LastTo, From: st.LastFrom, At: *st.LastAt, Notes: releaseNotes(st.LastTo, st)})
	}
	if st.ScheduledVersion != "" && st.ScheduledAt != nil && s.Auto {
		out = append(out, Notice{Kind: NoticeScheduled, Version: st.ScheduledVersion, At: *st.ScheduledAt, Notes: st.LatestNotes})
		return out
	}
	if Newer(st.LatestVersion, f.Current) {
		_, reason := Eligible(st, s, f)
		out = append(out, Notice{Kind: NoticeAvailable, Version: st.LatestVersion, Reason: reason, Notes: st.LatestNotes})
	}
	return out
}

func releaseNotes(version string, st db.UpdateState) string {
	if st.LatestVersion == version {
		return st.LatestNotes
	}
	return ""
}

const (
	OutcomeUpdated    = "updated"
	OutcomeRolledBack = "rolled-back"
	OutcomeFailed     = "failed"
)
