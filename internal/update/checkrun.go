package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

const (
	checkRunFile = "check.json"

	TimerQuiet  = time.Hour
	StartupCalm = 30 * time.Minute
)

type CheckRun struct {
	At             time.Time `json:"at"`
	Error          string    `json:"error,omitempty"`
	InstallVersion string    `json:"install_version,omitempty"`
	InstallError   string    `json:"install_error,omitempty"`
}

func CheckRunPath(dir string) string { return filepath.Join(dir, checkRunFile) }

func WriteCheckRun(dir string, run CheckRun) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, checkRunFile+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), CheckRunPath(dir))
}

func ReadCheckRun(dir string) (CheckRun, bool) {
	data, err := os.ReadFile(CheckRunPath(dir))
	if err != nil {
		return CheckRun{}, false
	}
	var run CheckRun
	if json.Unmarshal(data, &run) != nil || run.At.IsZero() {
		return CheckRun{}, false
	}
	return run, true
}

func RemoveCheckRun(dir string) {
	os.Remove(CheckRunPath(dir))
}

type CheckProblem struct {
	Kind    string
	Error   string
	Version string
}

const (
	ProblemFailed  = "failed"
	ProblemStopped = "stopped"
	ProblemFetch   = "fetch"
	ProblemInstall = "install"
)

func Problem(run CheckRun, seen bool, st db.UpdateState, s Settings, now, started time.Time) (CheckProblem, bool) {
	if !s.Check || !seen {
		return CheckProblem{}, false
	}
	switch {
	case run.Error != "":
		return CheckProblem{Kind: ProblemFailed, Error: run.Error}, true
	case run.InstallError != "" && !(st.LastOutcome == OutcomeRolledBack && st.LastTo == run.InstallVersion):
		return CheckProblem{Kind: ProblemInstall, Error: run.InstallError, Version: run.InstallVersion}, true
	case now.Sub(run.At) > TimerQuiet && now.Sub(started) > StartupCalm:
		return CheckProblem{Kind: ProblemStopped}, true
	case st.CheckError != "":
		return CheckProblem{Kind: ProblemFetch, Error: st.CheckError}, true
	}
	return CheckProblem{}, false
}
