package update

import (
	"errors"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		candidate, current string
		want               bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.1.0", "v1.0.9", true},
		{"v2.0.0", "v1.99.99", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "v1.0.0-rc.1", true},
		{"v1.0.0-rc.2", "v1.0.0-rc.1", true},
		{"v1.0.0-rc.10", "v1.0.0-rc.9", true},
		{"v1.0.0-rc.1", "v1.0.0", false},
		{"v1.0.1", "cf4ec6467c79", false},
		{"nonsense", "v1.0.0", false},
	}
	for _, tt := range cases {
		if got := Newer(tt.candidate, tt.current); got != tt.want {
			t.Errorf("Newer(%q, %q) = %t, want %t", tt.candidate, tt.current, got, tt.want)
		}
	}
}

func TestParseWindow(t *testing.T) {
	for _, bad := range []string{"", "03:00", "3-5", "25:00-26:00", "03:00-03:20"} {
		if _, err := ParseWindow(bad); err == nil {
			t.Errorf("ParseWindow(%q) err = nil, want an error", bad)
		}
	}
	w, err := ParseWindow("23:30-01:00")
	if err != nil {
		t.Fatalf("ParseWindow(across midnight) err = %v, want nil", err)
	}
	if w.length() != 90*time.Minute {
		t.Errorf("ParseWindow(23:30-01:00).length() = %s, want 1h30m", w.length())
	}
}

func TestNextCheckLandsWhereTheUpdateStillFits(t *testing.T) {
	w, _ := ParseWindow(DefaultWindow)
	zone := time.FixedZone("server", 9*3600)
	r := rand.New(rand.NewPCG(1, 2))
	cases := []struct {
		name string
		now  time.Time
		day  int
	}{
		{"before the window", time.Date(2026, 9, 13, 1, 0, 0, 0, zone), 13},
		{"inside the window", time.Date(2026, 9, 13, 3, 10, 0, 0, zone), 13},
		{"after the window", time.Date(2026, 9, 13, 12, 0, 0, 0, zone), 14},
		{"too late in the window", time.Date(2026, 9, 13, 4, 40, 0, 0, zone), 14},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 200; i++ {
				got := w.NextCheck(tt.now, r)
				earliest := time.Date(2026, 9, tt.day, 3, 0, 0, 0, zone)
				latest := time.Date(2026, 9, tt.day, 4, 30, 0, 0, zone)
				if got.Before(earliest) && !got.After(tt.now) || got.After(latest) || !got.After(tt.now) {
					t.Fatalf("NextCheck(%s) = %s, want after now and between %s and %s", tt.now, got, earliest, latest)
				}
			}
		})
	}
}

func eligibleState(now time.Time) db.UpdateState {
	published := now.Add(-48 * time.Hour)
	return db.UpdateState{LatestVersion: "v1.1.0", LatestPublishedAt: &published, LatestPostgres: "18.6.0"}
}

func defaultSettings() Settings {
	w, _ := ParseWindow(DefaultWindow)
	return Settings{Auto: true, PublicBanner: true, Check: true, Window: w, MinAge: DefaultMinAge}
}

func TestEligible(t *testing.T) {
	now := time.Date(2026, 9, 13, 3, 0, 0, 0, time.UTC)
	facts := Facts{Current: "v1.0.0", BundledPostgres: "18.6.0", Now: now}
	later := now.Add(time.Hour)
	fresh := now.Add(-time.Hour)

	cases := []struct {
		name   string
		change func(*db.UpdateState, *Settings, *Facts)
		want   string
	}{
		{"eligible", func(*db.UpdateState, *Settings, *Facts) {}, "v1.1.0"},
		{"auto off", func(_ *db.UpdateState, s *Settings, _ *Facts) { s.Auto = false }, ""},
		{"check off", func(_ *db.UpdateState, s *Settings, _ *Facts) { s.Check = false }, ""},
		{"container", func(_ *db.UpdateState, _ *Settings, f *Facts) { f.Container = true }, ""},
		{"development build", func(_ *db.UpdateState, _ *Settings, f *Facts) { f.Current = "cf4ec6467c79" }, ""},
		{"already newest", func(_ *db.UpdateState, _ *Settings, f *Facts) { f.Current = "v1.1.0" }, ""},
		{"pinned", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.PinnedVersion = "v1.0.0" }, ""},
		{"skipped", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.SkippedVersion = "v1.1.0" }, ""},
		{"skipped an older one", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.SkippedVersion = "v1.0.5" }, "v1.1.0"},
		{"failed before", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.FailedVersions = []string{"v1.1.0"} }, ""},
		{"postponed", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.PostponedUntil = &later }, ""},
		{"too fresh", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.LatestPublishedAt = &fresh }, ""},
		{"too fresh for a shorter age", func(st *db.UpdateState, s *Settings, _ *Facts) {
			st.LatestPublishedAt = &fresh
			s.MinAge = 30 * time.Minute
		}, "v1.1.0"},
		{"new PostgreSQL major", func(st *db.UpdateState, _ *Settings, _ *Facts) { st.LatestPostgres = "19.1.0" }, ""},
		{"own PostgreSQL ignores the major", func(st *db.UpdateState, _ *Settings, f *Facts) {
			st.LatestPostgres = "19.1.0"
			f.BundledPostgres = ""
		}, "v1.1.0"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			st, s, f := eligibleState(now), defaultSettings(), facts
			tt.change(&st, &s, &f)
			got, reason := Eligible(st, s, f)
			if got != tt.want {
				t.Errorf("Eligible() = %q (%s), want %q", got, reason, tt.want)
			}
			if got == "" && reason.String() == "" {
				t.Error("Eligible() reason = empty, want one when nothing is eligible")
			}
		})
	}
}

func TestTickSchedulesThenInstalls(t *testing.T) {
	zone := time.UTC
	r := rand.New(rand.NewPCG(3, 4))
	s := defaultSettings()
	published := time.Date(2026, 9, 10, 0, 0, 0, 0, zone)
	manifest := Manifest{Version: "v1.1.0", PublishedAt: published.Format(time.RFC3339), Postgres: "18.6.0"}
	fetches := 0
	fetch := func() (Manifest, error) { fetches++; return manifest, nil }
	facts := func(now time.Time) Facts { return Facts{Current: "v1.0.0", BundledPostgres: "18.6.0", Now: now} }

	var st db.UpdateState
	start := time.Date(2026, 9, 13, 12, 0, 0, 0, zone)
	if got := Tick(&st, s, facts(start), fetch, r); got != "" || fetches != 0 {
		t.Fatalf("first Tick() = %q with %d fetches, want nothing before the window", got, fetches)
	}
	check := *st.NextCheckAt
	if got := Tick(&st, s, facts(check.Add(-time.Minute)), fetch, r); got != "" || fetches != 0 {
		t.Fatalf("Tick() before the check = %q with %d fetches, want nothing", got, fetches)
	}
	if got := Tick(&st, s, facts(check), fetch, r); got != "" {
		t.Fatalf("Tick() at the check = %q, want it scheduled rather than installed", got)
	}
	if fetches != 1 || st.ScheduledVersion != "v1.1.0" || st.ScheduledAt == nil {
		t.Fatalf("Tick() at the check left fetches=%d scheduled=%q at=%v, want v1.1.0 scheduled", fetches, st.ScheduledVersion, st.ScheduledAt)
	}
	if want := check.Add(AnnounceAhead); !st.ScheduledAt.Equal(want) {
		t.Errorf("ScheduledAt = %s, want %s", st.ScheduledAt, want)
	}
	if at, ok := Banner(st, s, check.Add(time.Minute)); !ok || !at.Equal(*st.ScheduledAt) {
		t.Errorf("Banner() = %s, %t, want the scheduled time shown", at, ok)
	}
	if got := Tick(&st, s, facts(st.ScheduledAt.Add(-time.Second)), fetch, r); got != "" {
		t.Errorf("Tick() a second early = %q, want nothing", got)
	}
	if got := Tick(&st, s, facts(*st.ScheduledAt), fetch, r); got != "v1.1.0" {
		t.Errorf("Tick() when due = %q, want v1.1.0", got)
	}
}

func TestTickDropsASkippedOrMissedSchedule(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	s := defaultSettings()
	now := time.Date(2026, 9, 13, 3, 30, 0, 0, time.UTC)
	fetch := func() (Manifest, error) { return Manifest{}, errors.New("offline") }
	facts := func(at time.Time) Facts { return Facts{Current: "v1.0.0", Now: at} }

	st := eligibleState(now)
	next := now.Add(24 * time.Hour)
	st.NextCheckAt = &next
	at := now.Add(10 * time.Minute)
	st.ScheduledVersion, st.ScheduledAt = "v1.1.0", &at

	skipped := st
	Skip(&skipped)
	if got := Tick(&skipped, s, facts(at), fetch, r); got != "" || skipped.ScheduledVersion != "" {
		t.Errorf("Tick() after Skip = %q, scheduled %q, want nothing", got, skipped.ScheduledVersion)
	}

	missed := st
	if got := Tick(&missed, s, facts(at.Add(MissedAfter+time.Minute)), fetch, r); got != "" || missed.ScheduledAt != nil {
		t.Errorf("Tick() long after the schedule = %q, at %v, want it dropped", got, missed.ScheduledAt)
	}
}

func TestNoticesAfterARollback(t *testing.T) {
	now := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)
	at := now.Add(-time.Hour)
	st := db.UpdateState{LastOutcome: OutcomeRolledBack, LastFrom: "v1.0.0", LastTo: "v1.1.0", LastAt: &at, LastError: "health check failed"}
	got := Notices(st, defaultSettings(), Facts{Current: "v1.0.0", Now: now})
	if len(got) != 1 || got[0].Kind != NoticeRolledBack || got[0].Error != "health check failed" {
		t.Errorf("Notices() = %+v, want one rolled-back notice with the error", got)
	}
}
