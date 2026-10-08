package update

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestCheckRunRoundTrip(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 7, 1, 0, 0, 0, time.UTC)
	if err := WriteCheckRun(dir, CheckRun{At: at, Error: "permission denied"}); err != nil {
		t.Fatalf("WriteCheckRun() err = %v, want nil", err)
	}
	run, ok := ReadCheckRun(dir)
	if !ok {
		t.Fatalf("ReadCheckRun() ok = false, want true")
	}
	if !run.At.Equal(at) || run.Error != "permission denied" {
		t.Errorf("ReadCheckRun() = %+v, want at %v with the error", run, at)
	}
	RemoveCheckRun(dir)
	if _, ok := ReadCheckRun(dir); ok {
		t.Errorf("ReadCheckRun() after RemoveCheckRun ok = true, want false")
	}
}

func TestProblem(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	longAgo := now.Add(-24 * time.Hour)
	on := Settings{Check: true}
	cases := []struct {
		name    string
		run     CheckRun
		seen    bool
		st      db.UpdateState
		s       Settings
		started time.Time
		want    string
	}{
		{"no timer", CheckRun{}, false, db.UpdateState{CheckError: "x"}, on, longAgo, ""},
		{"checks off", CheckRun{At: now, Error: "boom"}, true, db.UpdateState{}, Settings{}, longAgo, ""},
		{"healthy", CheckRun{At: now.Add(-10 * time.Minute)}, true, db.UpdateState{}, on, longAgo, ""},
		{"tick failed", CheckRun{At: now, Error: "boom"}, true, db.UpdateState{}, on, longAgo, ProblemFailed},
		{"timer quiet", CheckRun{At: now.Add(-2 * time.Hour)}, true, db.UpdateState{}, on, longAgo, ProblemStopped},
		{"just started", CheckRun{At: now.Add(-2 * time.Hour)}, true, db.UpdateState{}, on, now.Add(-time.Minute), ""},
		{"fetch failed", CheckRun{At: now}, true, db.UpdateState{CheckError: "dial tcp"}, on, longAgo, ProblemFetch},
		{"install failed", CheckRun{At: now, InstallVersion: "v1.1.0", InstallError: "dial tcp"}, true, db.UpdateState{}, on, longAgo, ProblemInstall},
		{"install rolled back", CheckRun{At: now, InstallVersion: "v1.1.0", InstallError: "boom"}, true,
			db.UpdateState{LastOutcome: OutcomeRolledBack, LastTo: "v1.1.0"}, on, longAgo, ""},
	}
	for _, c := range cases {
		got, ok := Problem(c.run, c.seen, c.st, c.s, now, c.started)
		if c.want == "" && ok {
			t.Errorf("Problem(%s) = %+v, want none", c.name, got)
		}
		if c.want != "" && got.Kind != c.want {
			t.Errorf("Problem(%s).Kind = %q, want %q", c.name, got.Kind, c.want)
		}
	}
}
