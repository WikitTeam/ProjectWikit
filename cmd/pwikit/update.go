package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/entry"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/mail"
	"github.com/WikitTeam/ProjectWikit/internal/migrate"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/pgbundle"
	"github.com/WikitTeam/ProjectWikit/internal/service"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/update"
	"github.com/WikitTeam/ProjectWikit/internal/version"
	staticfiles "github.com/WikitTeam/ProjectWikit/static"
)

func updateUsage() {
	fmt.Fprint(os.Stderr, `Usage: pwikit update [check|status|rollback|unpin|postpone|skip] [options] [-- serve options]

  (none)    install the newest release now
  check     say whether a newer release exists
  status    show what the updater knows and has planned
  rollback  put back the release, and the data, from before the last update
  unpin     let automatic updates run again after pwikit update -to held a release
  postpone  put off the scheduled automatic update by 24 hours
  skip      never install the release that is scheduled or newest automatically

Options:
  -to        release to install instead of the newest; an older one is held until unpin
  -service   name of the installed service; defaults to pwikit
  -data-dir  state directory; defaults to the directory holding the executable
  -mirror    mirror to download from when GitHub cannot be reached
  -yes       answer yes to the rollback question

Options after -- are the ones the service was installed with, such as -database.
An instance installed as a system service is updated with sudo on Linux and macOS
and from a terminal opened with Run as administrator on Windows.
`)
}

type updateFlags struct {
	fs      *flag.FlagSet
	to      *string
	name    *string
	dataDir *string
	mirror  *string
	yes     *bool
	auto    *bool

	outcome *string
	from    *string
	target  *string
	kind    *string
	errText *string
	notify  *bool
	pin     *string
}

func newUpdateFlags(sub string) *updateFlags {
	fs := flag.NewFlagSet("update "+sub, flag.ContinueOnError)
	fs.Usage = updateUsage
	return &updateFlags{
		fs:      fs,
		to:      fs.String("to", "", "release to install instead of the newest"),
		name:    fs.String("service", service.DefaultName, "name of the installed service"),
		dataDir: fs.String("data-dir", "", "state directory; defaults to the directory holding the executable"),
		mirror:  fs.String("mirror", "", "mirror to download from when GitHub cannot be reached"),
		yes:     fs.Bool("yes", false, "answer yes to the rollback question"),
		auto:    fs.Bool("auto", false, "run as the scheduled update task"),
		outcome: fs.String("outcome", "", "what the update came to"),
		from:    fs.String("from", "", "release updated from"),
		target:  fs.String("target", "", "release updated to"),
		kind:    fs.String("kind", "", "kind of rollback point"),
		errText: fs.String("error", "", "why the update failed"),
		notify:  fs.Bool("notify", false, "mail the superusers"),
		pin:     fs.String("pin", "", "release to hold"),
	}
}

func updateCommand(args []string) error {
	sub := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub, args = args[0], args[1:]
	}
	serveArgs := []string{}
	if i := slices.Index(args, "--"); i >= 0 {
		serveArgs, args = args[i+1:], args[:i]
	}
	f := newUpdateFlags(sub)
	if err := f.fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.fs.NArg() > 0 {
		updateUsage()
		return fmt.Errorf("unexpected %q", f.fs.Arg(0))
	}
	p, err := paths.New(*f.dataDir)
	if err != nil {
		return err
	}

	switch sub {
	case "":
		if *f.auto {
			return autoUpdate(p, *f.name, serveArgs)
		}
		return manualUpdate(p, f, serveArgs)
	case "rollback":
		return rollbackUpdate(p, f, serveArgs)
	case "check":
		return checkUpdate(p, f, serveArgs)
	case "status", "unpin", "postpone", "skip":
		if handled, err := asOwner(p.Root()); handled || err != nil {
			return err
		}
		return editUpdateState(p, sub, serveArgs)
	case "tick":
		return tickUpdate(p, serveArgs)
	case "preflight":
		return preflightUpdate(p, serveArgs)
	case "record":
		return recordUpdate(p, f, serveArgs)
	}
	updateUsage()
	return fmt.Errorf("unknown update subcommand %q", sub)
}

type instanceSettings struct {
	opts     *serveOptions
	cfg      config.File
	settings update.Settings
}

func loadInstance(p *paths.Paths, serveArgs []string, mirror string) (instanceSettings, error) {
	o := newServeOptions()
	if err := o.fs.Parse(serveArgs); err != nil {
		return instanceSettings{}, fmt.Errorf("options after --: %w", err)
	}
	cfg, err := config.Load(p.Config())
	if err != nil {
		return instanceSettings{}, err
	}
	if _, err := o.resolve(cfg); err != nil {
		return instanceSettings{}, err
	}
	s, err := o.updateSettings(cfg)
	if err != nil {
		return instanceSettings{}, err
	}
	if mirror != "" {
		s.Mirror = mirror
	}
	return instanceSettings{opts: o, cfg: cfg, settings: s}, nil
}

func (in instanceSettings) databaseArgs() []string {
	if *in.opts.database == "" {
		return nil
	}
	return []string{"-database", *in.opts.database}
}

func (in instanceSettings) bundledPostgres() string {
	if *in.opts.database == "" {
		return pgbundle.Version
	}
	return ""
}

type updateLog struct {
	file *os.File
	out  io.Writer
}

func openUpdateLog(p *paths.Paths) *updateLog {
	l := &updateLog{out: os.Stdout}
	if err := os.MkdirAll(p.Logs(), 0o755); err == nil {
		if f, err := os.OpenFile(filepath.Join(p.Logs(), "update.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			l.file = f
			l.out = io.MultiWriter(os.Stdout, f)
		}
	}
	return l
}

func (l *updateLog) logf(format string, args ...any) {
	fmt.Fprintf(l.out, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func (l *updateLog) close(p *paths.Paths) {
	if l.file != nil {
		l.file.Close()
		update.ChownTree(l.file.Name(), p.Root())
	}
}

func runningExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

func autoUpdate(p *paths.Paths, name string, serveArgs []string) error {
	running, err := service.Running(name)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	exe, err := runningExecutable()
	if err != nil {
		return err
	}
	out, err := runOwner(context.Background(), p.Root(), exe, append([]string{"update", "tick", "-data-dir", p.Root(), "--"}, serveArgs...)...)
	if err != nil {
		l := openUpdateLog(p)
		defer l.close(p)
		l.logf("the scheduled check failed: %v: %s", err, strings.TrimSpace(string(out)))
		return err
	}
	var tick struct {
		Apply string `json:"apply"`
	}
	if err := json.Unmarshal(lastJSONLine(out), &tick); err != nil {
		return fmt.Errorf("read the scheduled check: %w", err)
	}
	if tick.Apply == "" {
		return nil
	}
	return installRelease(p, name, serveArgs, tick.Apply, "", true)
}

func manualUpdate(p *paths.Paths, f *updateFlags, serveArgs []string) error {
	in, err := loadInstance(p, serveArgs, *f.mirror)
	if err != nil {
		return err
	}
	target := *f.to
	if target != "" && !strings.HasPrefix(target, "v") {
		target = "v" + target
	}
	if target == "" {
		m, err := update.NewSource(in.settings.Mirror).Latest(context.Background())
		if err != nil {
			return err
		}
		target = m.Version
	}
	if target == version.String() {
		fmt.Printf("pwikit %s is already running\n", target)
		return nil
	}
	pin := ""
	if *f.to != "" && !update.Newer(target, version.String()) {
		pin = target
	}
	return installRelease(p, *f.name, serveArgs, target, pin, false)
}

func installRelease(p *paths.Paths, name string, serveArgs []string, target, pin string, auto bool) error {
	l := openUpdateLog(p)
	defer l.close(p)
	unlock, err := update.Lock(p.Updates())
	if err != nil {
		return err
	}
	defer unlock()

	in, err := loadInstance(p, serveArgs, "")
	if err != nil {
		return err
	}
	exe, err := runningExecutable()
	if err != nil {
		return err
	}
	running, err := service.Running(name)
	if err != nil && !auto {
		running = false
	}
	if !running && serveAlive(p) {
		return errors.New("pwikit serve is running in the foreground; stop it first, or update an instance installed with pwikit service install")
	}

	ctx := context.Background()
	src := update.NewSource(in.settings.Mirror)
	src.Log = func(line string) { l.logf("%s", line) }
	sums, err := src.Checksums(ctx, target)
	if err != nil {
		return fmt.Errorf("read the checksums of %s: %w", target, err)
	}
	file := update.PackageFile(target, runtime.GOOS, runtime.GOARCH)
	sum, ok := sums[file]
	if !ok {
		return fmt.Errorf("release %s has no package for %s/%s", target, runtime.GOOS, runtime.GOARCH)
	}

	l.logf("updating %s from %s to %s", p.Root(), version.String(), target)
	applier := &update.Applier{
		Root:       p.Root(),
		PGData:     p.PGData(),
		Executable: exe,
		Bundled:    in.bundledPostgres() != "",
		Current:    version.String(),
		Database:   in.databaseArgs(),
		Source:     src,
		Machine:    serviceMachine{name: name, stopped: !running},
		Run: func(ctx context.Context, exe string, args ...string) ([]byte, error) {
			out, err := runOwner(ctx, p.Root(), exe, args...)
			if len(out) > 0 {
				l.logf("%s", strings.TrimSpace(string(out)))
			}
			return out, err
		},
		Maintenance: func(ctx context.Context) (func(), error) { return maintenance(p) },
		Logf:        l.logf,
	}
	if !running {
		applier.Maintenance = nil
		applier.Offline = true
	}
	outcome := applier.Apply(ctx, update.Target{Version: target, File: file, SHA256: sum, AllowPostgresMove: !auto})

	record := []string{"update", "record", "-data-dir", p.Root(), "-from", outcome.From, "-target", outcome.To, "-kind", outcome.Kind}
	switch {
	case outcome.Err == nil:
		record = append(record, "-outcome", update.OutcomeUpdated)
		if pin != "" {
			record = append(record, "-pin", pin)
		}
	case outcome.RolledBack:
		record = append(record, "-outcome", update.OutcomeRolledBack, "-error", outcome.Err.Error(), "-notify")
	default:
		record = append(record, "-outcome", update.OutcomeFailed, "-error", outcome.Err.Error())
	}
	if running || outcome.Err == nil {
		if out, err := runOwner(ctx, p.Root(), exe, append(append(record, "--"), serveArgs...)...); err != nil {
			l.logf("could not record the outcome: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}

	if outcome.Err == nil {
		if !running {
			fmt.Printf("installed %s; start pwikit serve again\n", target)
		}
		return nil
	}
	if outcome.RollbackErr != nil {
		return fmt.Errorf("update to %s failed: %v; rolling back failed as well: %v. See %s",
			target, outcome.Err, outcome.RollbackErr, filepath.Join(p.Logs(), "update.log"))
	}
	if outcome.RolledBack {
		return fmt.Errorf("update to %s failed and %s was put back: %v", target, outcome.From, outcome.Err)
	}
	return fmt.Errorf("update to %s failed: %v", target, outcome.Err)
}

func rollbackUpdate(p *paths.Paths, f *updateFlags, serveArgs []string) error {
	point, err := update.ReadRollback(update.RollbackDir(p.Root()))
	if err != nil {
		return err
	}
	if point.Expired(time.Now()) {
		return fmt.Errorf("the rollback point from %s is older than seven days and is no longer used", point.CreatedAt.Format(time.RFC3339))
	}
	if point.From == version.String() {
		return fmt.Errorf("pwikit %s is already running", point.From)
	}
	if point.Kind != update.RollbackNone && !*f.yes {
		fmt.Printf("This puts back pwikit %s and the database as it was at %s.\n", point.From, point.CreatedAt.Local().Format("2006-01-02 15:04"))
		fmt.Print("Everything written to the site since then is lost. Continue? [y/N] ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			return errors.New("nothing was changed")
		}
	}

	l := openUpdateLog(p)
	defer l.close(p)
	unlock, err := update.Lock(p.Updates())
	if err != nil {
		return err
	}
	defer unlock()
	in, err := loadInstance(p, serveArgs, "")
	if err != nil {
		return err
	}
	exe, err := runningExecutable()
	if err != nil {
		return err
	}
	running, _ := service.Running(*f.name)
	if !running && serveAlive(p) {
		return errors.New("pwikit serve is running in the foreground; stop it first")
	}
	applier := &update.Applier{
		Root: p.Root(), PGData: p.PGData(), Executable: exe,
		Bundled: in.bundledPostgres() != "", Current: version.String(), Database: in.databaseArgs(),
		Machine: serviceMachine{name: *f.name, stopped: !running},
		Run: func(ctx context.Context, exe string, args ...string) ([]byte, error) {
			return runOwner(ctx, p.Root(), exe, args...)
		},
		Maintenance: func(ctx context.Context) (func(), error) { return maintenance(p) },
		Logf:        l.logf,
	}
	if !running {
		applier.Maintenance = nil
		applier.Offline = true
	}
	if _, err := applier.Rollback(context.Background()); err != nil {
		return err
	}
	record := []string{"update", "record", "-data-dir", p.Root(), "-outcome", "manual-rollback",
		"-from", point.To, "-target", point.From, "-pin", point.From, "--"}
	if out, err := runOwner(context.Background(), p.Root(), exe, append(record, serveArgs...)...); err != nil {
		l.logf("could not record the rollback: %v: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Printf("put back pwikit %s; automatic updates stay off until pwikit update unpin\n", point.From)
	return nil
}

func checkUpdate(p *paths.Paths, f *updateFlags, serveArgs []string) error {
	in, err := loadInstance(p, serveArgs, *f.mirror)
	if err != nil {
		return err
	}
	src := update.NewSource(in.settings.Mirror)
	src.Log = func(line string) { fmt.Fprintln(os.Stderr, line) }
	m, err := src.Latest(context.Background())
	if err != nil {
		return err
	}
	current := version.String()
	fmt.Printf("running   %s\n", current)
	fmt.Printf("newest    %s, released %s\n", m.Version, m.Published().Local().Format("2006-01-02 15:04"))
	if !update.Newer(m.Version, current) {
		fmt.Println("nothing newer to install")
		return nil
	}
	if update.PostgresMajor(m.Postgres) != update.PostgresMajor(pgbundle.Version) && in.bundledPostgres() != "" {
		fmt.Printf("it moves the bundled PostgreSQL to version %s, which takes a backup and a restore\n", update.PostgresMajor(m.Postgres))
	}
	fmt.Printf("notes     %s\n", m.Notes)
	fmt.Println("install it with: pwikit update")
	return nil
}

func tickUpdate(p *paths.Paths, serveArgs []string) error {
	in, err := loadInstance(p, serveArgs, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn, release, err := openInstanceDB(ctx, p, in)
	if err != nil {
		return err
	}
	defer release()
	st, err := conn.UpdateState(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	if st.RollbackExpiresAt != nil && now.After(*st.RollbackExpiresAt) {
		os.RemoveAll(update.RollbackDir(p.Root()))
		st.RollbackVersion, st.RollbackKind, st.RollbackExpiresAt = "", "", nil
	}
	src := update.NewSource(in.settings.Mirror)
	facts := update.Facts{Current: version.String(), BundledPostgres: in.bundledPostgres(), Container: inContainer(), Now: now}
	apply := update.Tick(&st, in.settings, facts, func() (update.Manifest, error) { return src.Latest(ctx) },
		rand.New(rand.NewPCG(uint64(now.UnixNano()), uint64(os.Getpid()))))
	if err := conn.SaveUpdateState(ctx, st); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"apply": apply})
}

func preflightUpdate(p *paths.Paths, serveArgs []string) error {
	pre := update.Preflight{Version: version.String(), Postgres: pgbundle.Version}
	defer func() { json.NewEncoder(os.Stdout).Encode(pre) }()

	in, err := loadInstance(p, serveArgs, "")
	if err != nil {
		pre.Problem = err.Error()
		return nil
	}
	if in.bundledPostgres() != "" {
		if have, err := os.ReadFile(filepath.Join(p.PGData(), "PG_VERSION")); err == nil {
			pre.PostgresMoves = strings.TrimSpace(string(have)) != update.PostgresMajor(pgbundle.Version)
		}
	}
	ctx := context.Background()
	dsn, release, err := resolveDatabase(ctx, *in.opts.database, p.Root())
	if err != nil {
		pre.Problem = err.Error()
		return nil
	}
	defer release()
	state, err := migrate.Status(ctx, dsn)
	if err != nil {
		pre.Problem = err.Error()
		return nil
	}
	pre.Pending = state.Pending
	pre.PendingBreaking = state.PendingBreaking()
	switch {
	case len(state.UnknownBreaking) > 0:
		pre.Problem = (&migrate.NewerSchemaError{Migrations: state.UnknownBreaking, AppliedBy: state.AppliedBy}).Error()
	case len(state.Unknown) > 0 && len(state.Pending) > 0:
		pre.Problem = fmt.Sprintf("the database holds %s from another release while this one still has %s to apply",
			strings.Join(state.Unknown, ", "), strings.Join(state.Pending, ", "))
	}
	return nil
}

func recordUpdate(p *paths.Paths, f *updateFlags, serveArgs []string) error {
	in, err := loadInstance(p, serveArgs, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn, release, err := openInstanceDB(ctx, p, in)
	if err != nil {
		return err
	}
	defer release()
	st, err := conn.UpdateState(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	st.LastOutcome, st.LastFrom, st.LastTo, st.LastError, st.LastAt = *f.outcome, *f.from, *f.target, *f.errText, &now
	switch *f.outcome {
	case update.OutcomeUpdated:
		expires := now.Add(update.RollbackKept)
		st.RollbackVersion, st.RollbackKind, st.RollbackExpiresAt = *f.from, *f.kind, &expires
		st.ScheduledVersion, st.ScheduledAt = "", nil
		st.PinnedVersion = ""
	case update.OutcomeRolledBack:
		if !slices.Contains(st.FailedVersions, *f.target) {
			st.FailedVersions = append(st.FailedVersions, *f.target)
		}
		st.ScheduledVersion, st.ScheduledAt = "", nil
		st.RollbackVersion, st.RollbackKind, st.RollbackExpiresAt = "", "", nil
	case "manual-rollback":
		st.RollbackVersion, st.RollbackKind, st.RollbackExpiresAt = "", "", nil
		st.ScheduledVersion, st.ScheduledAt = "", nil
	}
	if *f.pin != "" {
		st.PinnedVersion = *f.pin
	}
	if err := conn.SaveUpdateState(ctx, st); err != nil {
		return err
	}
	if *f.notify {
		notifyRollback(ctx, p, in, conn, st)
	}
	return nil
}

func notifyRollback(ctx context.Context, p *paths.Paths, in instanceSettings, conn *db.DB, st db.UpdateState) {
	to, err := conn.SuperuserEmails(ctx)
	if err != nil || len(to) == 0 {
		return
	}
	bundle, err := i18n.Load(p.Locales())
	if err != nil {
		return
	}
	loc := bundle.Localizer(i18n.DefaultLanguage)
	at := ""
	if st.LastAt != nil {
		at = st.LastAt.Local().Format("2006-01-02 15:04 MST")
	}
	subject := loc.T("update.mail-rolled-back-subject", "version", st.LastTo, "from", st.LastFrom)
	body := loc.T("update.mail-rolled-back-body", "root", p.Root(), "time", at, "from", st.LastFrom,
		"version", st.LastTo, "error", st.LastError, "log", filepath.Join(p.Logs(), "update.log"))
	if err := mail.New(mailConfig(in.cfg.Mail)).Send(ctx, to, subject, body); err != nil {
		fmt.Fprintf(os.Stderr, "could not mail the superusers about the rollback: %v\n", err)
	}
}

func editUpdateState(p *paths.Paths, sub string, serveArgs []string) error {
	in, err := loadInstance(p, serveArgs, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn, release, err := openInstanceDB(ctx, p, in)
	if err != nil {
		return err
	}
	defer release()
	st, err := conn.UpdateState(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	switch sub {
	case "status":
		printUpdateStatus(st, in, now)
		return nil
	case "unpin":
		if st.PinnedVersion == "" {
			fmt.Println("no release is held")
			return nil
		}
		fmt.Printf("released the hold on %s\n", st.PinnedVersion)
		st.PinnedVersion = ""
	case "postpone":
		update.Postpone(&st, now)
		fmt.Printf("automatic updates wait until %s\n", st.PostponedUntil.Local().Format("2006-01-02 15:04"))
	case "skip":
		if skipped := update.Skip(&st); skipped != "" {
			fmt.Printf("%s will not be installed automatically\n", skipped)
		}
	}
	return conn.SaveUpdateState(ctx, st)
}

func printUpdateStatus(st db.UpdateState, in instanceSettings, now time.Time) {
	stamp := func(at *time.Time) string {
		if at == nil {
			return "never"
		}
		return at.Local().Format("2006-01-02 15:04")
	}
	facts := update.Facts{Current: version.String(), BundledPostgres: in.bundledPostgres(), Container: inContainer(), Now: now}
	fmt.Printf("running      %s\n", version.String())
	fmt.Printf("automatic    %t, window %s, releases older than %s\n", in.settings.Auto, in.settings.Window, in.settings.MinAge)
	fmt.Printf("checked      %s\n", stamp(st.CheckedAt))
	if st.CheckError != "" {
		fmt.Printf("check error  %s\n", st.CheckError)
	}
	fmt.Printf("next check   %s\n", stamp(st.NextCheckAt))
	if st.LatestVersion != "" {
		fmt.Printf("newest       %s, released %s\n", st.LatestVersion, stamp(st.LatestPublishedAt))
	}
	if st.ScheduledVersion != "" {
		fmt.Printf("scheduled    %s at %s\n", st.ScheduledVersion, stamp(st.ScheduledAt))
	} else if _, reason := update.Eligible(st, in.settings, facts); reason.Code != "" {
		fmt.Printf("not planned  %s\n", reason)
	}
	if st.PinnedVersion != "" {
		fmt.Printf("held at      %s\n", st.PinnedVersion)
	}
	if st.LastOutcome != "" {
		fmt.Printf("last update  %s from %s to %s at %s\n", st.LastOutcome, st.LastFrom, st.LastTo, stamp(st.LastAt))
		if st.LastError != "" {
			fmt.Printf("last error   %s\n", st.LastError)
		}
	}
	if st.RollbackVersion != "" {
		fmt.Printf("rollback     to %s, kept until %s\n", st.RollbackVersion, stamp(st.RollbackExpiresAt))
	}
}

func openInstanceDB(ctx context.Context, p *paths.Paths, in instanceSettings) (*db.DB, func(), error) {
	dsn, release, err := resolveDatabase(ctx, *in.opts.database, p.Root())
	if err != nil {
		return nil, nil, err
	}
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		release()
		return nil, nil, err
	}
	return conn, func() { conn.Close(); release() }, nil
}

type serviceMachine struct {
	name    string
	stopped bool
}

func (m serviceMachine) StopService(context.Context) error {
	if m.stopped {
		return nil
	}
	return service.Stop(m.name)
}

func (m serviceMachine) StartService(context.Context) error {
	if m.stopped {
		return nil
	}
	return service.Start(m.name)
}

func maintenance(p *paths.Paths) (func(), error) {
	rt, err := update.ReadRuntime(p.Updates())
	if err != nil {
		return nil, err
	}
	bundle, err := i18n.Load(p.Locales())
	if err != nil {
		return nil, err
	}
	var assets *static.Assets
	var files http.Handler
	if staticfiles.Embedded {
		assets = static.NewAssets(staticfiles.Files)
		files = static.New(staticfiles.Files, http.NotFoundHandler())
	} else {
		assets = static.NewAssets(nil)
	}
	allowed := rt.Hosts
	cfg := entry.Config{
		Mode:      entry.Mode(rt.Mode),
		Plain:     rt.Plain,
		Secure:    rt.Secure,
		CertFile:  rt.CertFile,
		KeyFile:   rt.KeyFile,
		CacheDir:  rt.CacheDir,
		Email:     rt.Email,
		Directory: rt.Directory,
		Hosts: func(_ context.Context, host string) error {
			if slices.Contains(allowed, host) {
				return nil
			}
			return fmt.Errorf("host %q is not a site on this server", host)
		},
		Handler: update.MaintenanceHandler(bundle, assets, files),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- entry.Serve(ctx, cfg) }()
	select {
	case err := <-done:
		cancel()
		return nil, err
	case <-time.After(500 * time.Millisecond):
	}
	return func() {
		cancel()
		<-done
		if rt.CacheDir != "" {
			update.ChownTree(rt.CacheDir, p.Root())
		}
	}, nil
}

func serveAlive(p *paths.Paths) bool {
	rt, err := update.ReadRuntime(p.Updates())
	if err != nil || rt.PID == 0 || rt.PID == os.Getpid() {
		return false
	}
	return processAlive(rt.PID)
}

func lastJSONLine(out []byte) []byte {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); strings.HasPrefix(line, "{") {
			return []byte(line)
		}
	}
	return nil
}
