package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	RollbackNone    = "binary"
	RollbackPGData  = "pgdata"
	RollbackBackup  = "backup"
	rollbackDir     = "rollback"
	stagingDir      = "staging"
	failedDir       = "failed"
	rollbackInfo    = "rollback.json"
	databaseBackup  = "database.pwbak"
	postgresBackup  = "postgres-move.pwbak"
	diskHeadroom    = 256 << 20
	defaultHealthIn = 15 * time.Minute
)

type Machine interface {
	StopService(ctx context.Context) error
	StartService(ctx context.Context) error
}

type Runner func(ctx context.Context, exe string, args ...string) ([]byte, error)

type Target struct {
	Version string
	File    string
	SHA256  string

	AllowPostgresMove bool
}

type Preflight struct {
	Version         string   `json:"version"`
	Postgres        string   `json:"postgres"`
	Pending         []string `json:"pending"`
	PendingBreaking bool     `json:"pending_breaking"`
	PostgresMoves   bool     `json:"postgres_moves"`
	Problem         string   `json:"problem,omitempty"`
}

type RollbackPoint struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

func (p RollbackPoint) Expired(now time.Time) bool {
	return now.Sub(p.CreatedAt) > RollbackKept
}

type Outcome struct {
	From        string
	To          string
	Kind        string
	Err         error
	RolledBack  bool
	RollbackErr error
}

type Applier struct {
	Root       string
	PGData     string
	Executable string
	Bundled    bool
	Current    string
	Database   []string

	Offline bool

	Source      Source
	Machine     Machine
	Run         Runner
	Maintenance func(ctx context.Context) (stop func(), err error)
	Logf        func(format string, args ...any)

	HealthWithin time.Duration
}

func (a *Applier) dir(parts ...string) string {
	return filepath.Join(append([]string{a.Root, "update"}, parts...)...)
}

func (a *Applier) logf(format string, args ...any) {
	if a.Logf != nil {
		a.Logf(format, args...)
	}
}

func (a *Applier) healthWithin() time.Duration {
	if a.HealthWithin > 0 {
		return a.HealthWithin
	}
	return defaultHealthIn
}

func (a *Applier) Apply(ctx context.Context, t Target) Outcome {
	out := Outcome{From: a.Current, To: t.Version}
	fail := func(err error) Outcome {
		a.logf("update to %s stopped before the site was touched: %v", t.Version, err)
		out.Err = err
		return out
	}

	staging := a.dir(stagingDir)
	if err := os.RemoveAll(staging); err != nil {
		return fail(err)
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return fail(err)
	}
	pkg := filepath.Join(staging, t.File)
	a.logf("downloading %s", t.File)
	if err := a.Source.Download(ctx, t.Version, t.File, pkg, t.SHA256); err != nil {
		return fail(err)
	}
	next := filepath.Join(staging, ExecutableName(runtime.GOOS))
	if err := ExtractExecutable(pkg, next, runtime.GOOS); err != nil {
		return fail(err)
	}
	if err := ChownTree(a.dir(), a.Root); err != nil {
		return fail(err)
	}

	pre, err := a.preflight(ctx, next)
	if err != nil {
		return fail(err)
	}
	switch {
	case pre.Version != t.Version:
		return fail(fmt.Errorf("the downloaded program says it is %s, not %s", pre.Version, t.Version))
	case pre.Problem != "":
		return fail(errors.New(pre.Problem))
	case pre.PostgresMoves && !t.AllowPostgresMove:
		return fail(fmt.Errorf("%s moves the bundled PostgreSQL to version %s; run pwikit update by hand", t.Version, PostgresMajor(pre.Postgres)))
	}

	kind := RollbackNone
	if pre.PendingBreaking || pre.PostgresMoves {
		kind = RollbackBackup
		if a.Bundled {
			kind = RollbackPGData
		}
	}
	if kind == RollbackPGData && !pre.PostgresMoves {
		if err := a.roomFor(a.PGData); err != nil {
			return fail(err)
		}
	}
	out.Kind = kind

	point := a.dir(rollbackDir)
	if err := os.RemoveAll(point); err != nil {
		return fail(err)
	}
	if err := os.MkdirAll(point, 0o755); err != nil {
		return fail(err)
	}
	if err := CopyFile(a.Executable, filepath.Join(point, ExecutableName(runtime.GOOS))); err != nil {
		return fail(err)
	}
	if pre.PostgresMoves {
		a.logf("backing up the database before PostgreSQL moves to version %s", PostgresMajor(pre.Postgres))
		if _, err := a.run(ctx, a.Executable, append([]string{"backup", "create", "-output", filepath.Join(point, postgresBackup)}, a.base()...)...); err != nil {
			return fail(fmt.Errorf("back up the database: %w", err))
		}
	}
	if err := writeRollback(point, RollbackPoint{From: a.Current, To: t.Version, Kind: kind, CreatedAt: time.Now().UTC()}); err != nil {
		return fail(err)
	}
	if err := ChownTree(a.dir(), a.Root); err != nil {
		return fail(err)
	}

	a.logf("stopping the service to install %s", t.Version)
	if err := a.Machine.StopService(ctx); err != nil {
		return fail(fmt.Errorf("stop the service: %w", err))
	}
	stopMaintenance := a.maintenance(ctx)

	replaced, cause := a.swap(ctx, kind, pre, point, next)
	stopMaintenance()
	if cause != nil && !replaced {
		a.logf("update to %s stopped before the new program was put in place: %v", t.Version, cause)
		out.Err = cause
		if !a.Offline {
			since := time.Now()
			if err := a.Machine.StartService(ctx); err != nil {
				out.RollbackErr = fmt.Errorf("start %s again: %w", a.Current, err)
			} else if _, err := WaitHealthy(ctx, a.dir(), a.Current, since, a.healthWithin()); err != nil {
				out.RollbackErr = err
			}
		}
		return out
	}
	if cause == nil && a.Offline {
		os.RemoveAll(staging)
		a.logf("installed %s; it runs once pwikit is started again", t.Version)
		return out
	}
	if cause == nil {
		since := time.Now()
		a.logf("starting %s", t.Version)
		if err := a.Machine.StartService(ctx); err != nil {
			cause = fmt.Errorf("start the service: %w", err)
		} else if _, err := WaitHealthy(ctx, a.dir(), t.Version, since, a.healthWithin()); err != nil {
			cause = err
		}
	}
	if cause == nil {
		os.RemoveAll(staging)
		a.logf("updated from %s to %s", a.Current, t.Version)
		return out
	}

	a.logf("update to %s failed: %v", t.Version, cause)
	out.Err = cause
	out.RolledBack = true
	out.RollbackErr = a.restore(ctx, RollbackPoint{From: a.Current, To: t.Version, Kind: kind}, point)
	return out
}

// Until the new program is in place nothing is lost by stopping, so a failure
// before that undoes its own step and leaves the data exactly as it was.
func (a *Applier) swap(ctx context.Context, kind string, pre Preflight, point, next string) (bool, error) {
	saved := filepath.Join(point, "pgdata")
	moved := false
	undo := func() {
		if moved {
			if err := os.Rename(saved, a.PGData); err != nil {
				a.logf("could not move pgdata back from %s: %v", saved, err)
			}
			return
		}
		os.RemoveAll(saved)
	}
	switch {
	case kind == RollbackPGData && pre.PostgresMoves:
		a.logf("moving the PostgreSQL %s data aside", a.CurrentPostgres())
		if err := os.Rename(a.PGData, saved); err != nil {
			return false, fmt.Errorf("move pgdata aside: %w", err)
		}
		moved = true
	case kind == RollbackPGData:
		a.logf("copying pgdata as the point to roll back to")
		if err := CopyTree(a.PGData, saved); err != nil {
			undo()
			return false, fmt.Errorf("copy pgdata: %w", err)
		}
	case kind == RollbackBackup:
		a.logf("backing up the database as the point to roll back to")
		if _, err := a.run(ctx, a.Executable, append([]string{"backup", "create", "-no-files", "-output", filepath.Join(point, databaseBackup)}, a.base()...)...); err != nil {
			return false, fmt.Errorf("back up the database: %w", err)
		}
	}
	if err := ChownTree(a.dir(), a.Root); err != nil {
		undo()
		return false, err
	}
	if err := Replace(a.Executable, next, a.dir(stagingDir, "replaced-"+ExecutableName(runtime.GOOS))); err != nil {
		undo()
		return false, err
	}
	if err := ChownTree(a.Executable, a.Root); err != nil {
		return true, err
	}
	if pre.PostgresMoves {
		a.logf("restoring the database into PostgreSQL %s", PostgresMajor(pre.Postgres))
		args := append([]string{"backup", "restore", filepath.Join(point, postgresBackup), "-force", "-no-safety-backup"}, a.base()...)
		if _, err := a.run(ctx, a.Executable, args...); err != nil {
			return true, fmt.Errorf("restore into the new PostgreSQL: %w", err)
		}
	}
	return true, nil
}

func (a *Applier) Rollback(ctx context.Context) (RollbackPoint, error) {
	point := a.dir(rollbackDir)
	p, err := ReadRollback(point)
	if err != nil {
		return p, err
	}
	if err := a.Machine.StopService(ctx); err != nil {
		return p, fmt.Errorf("stop the service: %w", err)
	}
	return p, a.restore(ctx, p, point)
}

func (a *Applier) restore(ctx context.Context, p RollbackPoint, point string) error {
	a.logf("rolling back to %s", p.From)
	a.Machine.StopService(ctx)
	stopMaintenance := a.maintenance(ctx)

	err := a.putBack(ctx, p, point)
	stopMaintenance()
	if err != nil {
		a.logf("rolling back failed: %v", err)
		return err
	}
	if a.Offline {
		a.logf("put back %s; it runs once pwikit is started again", p.From)
		return nil
	}
	since := time.Now()
	if err := a.Machine.StartService(ctx); err != nil {
		return fmt.Errorf("start %s again: %w", p.From, err)
	}
	if _, err := WaitHealthy(ctx, a.dir(), p.From, since, a.healthWithin()); err != nil {
		return fmt.Errorf("%s did not come back healthy: %w", p.From, err)
	}
	a.logf("rolled back to %s", p.From)
	return nil
}

func (a *Applier) putBack(ctx context.Context, p RollbackPoint, point string) error {
	old := filepath.Join(point, ExecutableName(runtime.GOOS))
	if _, err := os.Stat(old); err != nil {
		return fmt.Errorf("the rollback point holds no program: %w", err)
	}
	failed := a.dir(failedDir, time.Now().UTC().Format("20060102-150405"))
	if err := os.MkdirAll(failed, 0o755); err != nil {
		return err
	}
	switch p.Kind {
	case RollbackPGData:
		saved := filepath.Join(point, "pgdata")
		if _, err := os.Stat(saved); err != nil {
			return fmt.Errorf("the rollback point holds no pgdata: %w", err)
		}
		if _, err := os.Stat(a.PGData); err == nil {
			if err := os.Rename(a.PGData, filepath.Join(failed, "pgdata")); err != nil {
				return fmt.Errorf("move the updated pgdata aside: %w", err)
			}
		}
		if err := os.Rename(saved, a.PGData); err != nil {
			return fmt.Errorf("put pgdata back: %w", err)
		}
	}
	running := filepath.Join(point, "run-"+ExecutableName(runtime.GOOS))
	if err := CopyFile(old, running); err != nil {
		return err
	}
	if err := Replace(a.Executable, running, filepath.Join(failed, ExecutableName(runtime.GOOS))); err != nil {
		return err
	}
	for _, path := range []string{a.Executable, a.PGData, a.dir()} {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := ChownTree(path, a.Root); err != nil {
			return err
		}
	}
	if p.Kind == RollbackBackup {
		args := append([]string{"backup", "restore", filepath.Join(point, databaseBackup), "-force", "-no-safety-backup", "-no-files"}, a.base()...)
		if _, err := a.run(ctx, a.Executable, args...); err != nil {
			return fmt.Errorf("restore the database: %w", err)
		}
	}
	return nil
}

func (a *Applier) preflight(ctx context.Context, next string) (Preflight, error) {
	body, err := a.run(ctx, next, append([]string{"update", "preflight"}, a.base()...)...)
	if err != nil {
		return Preflight{}, fmt.Errorf("check the new release against this instance: %w", err)
	}
	var pre Preflight
	if err := json.Unmarshal(lastLine(body), &pre); err != nil {
		return Preflight{}, fmt.Errorf("read what the new release reported: %w", err)
	}
	return pre, nil
}

func (a *Applier) base() []string {
	return append([]string{"-data-dir", a.Root}, a.Database...)
}

func (a *Applier) run(ctx context.Context, exe string, args ...string) ([]byte, error) {
	a.logf("running %s %v", filepath.Base(exe), args)
	return a.Run(ctx, exe, args...)
}

func (a *Applier) maintenance(ctx context.Context) func() {
	if a.Maintenance == nil {
		return func() {}
	}
	stop, err := a.Maintenance(ctx)
	if err != nil {
		a.logf("the maintenance page could not start, so visitors see no answer meanwhile: %v", err)
		return func() {}
	}
	return stop
}

func (a *Applier) roomFor(dir string) error {
	size, err := TreeSize(dir)
	if err != nil {
		return fmt.Errorf("measure %s: %w", dir, err)
	}
	free, err := FreeBytes(a.Root)
	if err != nil {
		return fmt.Errorf("measure the free space: %w", err)
	}
	if need := uint64(size) + diskHeadroom; free < need {
		return fmt.Errorf("the rollback point needs %d MB and only %d MB are free", need>>20, free>>20)
	}
	return nil
}

func (a *Applier) CurrentPostgres() string {
	body, err := os.ReadFile(filepath.Join(a.PGData, "PG_VERSION"))
	if err != nil {
		return ""
	}
	return string(bytesTrim(body))
}

func writeRollback(dir string, p RollbackPoint) error {
	body, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, rollbackInfo), body, 0o644)
}

func ReadRollback(dir string) (RollbackPoint, error) {
	body, err := os.ReadFile(filepath.Join(dir, rollbackInfo))
	if errors.Is(err, os.ErrNotExist) {
		return RollbackPoint{}, errors.New("there is no rollback point; one is kept for seven days after an update")
	}
	if err != nil {
		return RollbackPoint{}, err
	}
	var p RollbackPoint
	if err := json.Unmarshal(body, &p); err != nil {
		return RollbackPoint{}, fmt.Errorf("read %s: %w", rollbackInfo, err)
	}
	return p, nil
}

func RollbackDir(root string) string { return filepath.Join(root, "update", rollbackDir) }

func lastLine(body []byte) []byte {
	trimmed := bytesTrim(body)
	for i := len(trimmed) - 1; i >= 0; i-- {
		if trimmed[i] == '\n' {
			return trimmed[i+1:]
		}
	}
	return trimmed
}

func bytesTrim(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}
