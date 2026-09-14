package pgbundle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

const Database = "pwikit"

const (
	postmasterFile = "postmaster.pid"
	startLogFile   = "postgresql-start.log"
	toolLogFile    = "postgresql-tools.log"

	progressEvery = 15 * time.Second
)

type Config struct {
	Root     string
	Postgres string
	Data     string
	Secrets  string
	Logs     string
	Log      *slog.Logger
}

func (c Config) layout() Layout {
	return NewLayout(c.Root, c.Postgres, c.Data, c.Secrets)
}

func (c Config) logger() *slog.Logger {
	if c.Log != nil {
		return c.Log
	}
	return slog.Default()
}

type Server struct {
	cfg    Config
	layout Layout
	plan   Plan
	lock   *Lock
	proc   *process
	exited chan struct{}
	err    error
}

func Start(ctx context.Context, cfg Config, owner Owner) (_ *Server, err error) {
	if err := refuseSuperuser(); err != nil {
		return nil, err
	}
	for _, dir := range []string{cfg.Root, cfg.Logs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", dir, err)
		}
	}
	lock, err := TryLock(filepath.Join(cfg.Root, lockFile), owner)
	if err != nil {
		return nil, err
	}
	s := &Server{cfg: cfg, layout: cfg.layout(), lock: lock, exited: make(chan struct{})}
	defer func() {
		if err != nil {
			s.abandon()
		}
	}()

	log := cfg.logger()
	if err := s.clearLeftover(ctx); err != nil {
		return nil, err
	}
	if unpacked, err := Unpack(cfg.Postgres); err != nil {
		return nil, err
	} else if unpacked {
		log.Info("pwikit unpacked its PostgreSQL", "version", Version, "dir", cfg.Postgres)
	}
	if err := s.layout.Locate(runtime.GOOS); err != nil {
		if !Embedded() {
			return nil, fmt.Errorf("this pwikit was built without a PostgreSQL of its own and found none in %s.\n"+
				"  Point it at a PostgreSQL you run yourself with -database or DATABASE_URL", cfg.Postgres)
		}
		return nil, err
	}

	binaryMajor, err := s.binaryMajor(ctx)
	if err != nil {
		return nil, err
	}
	dataMajor, err := DataVersion(cfg.Data)
	if err != nil {
		return nil, err
	}
	if err := Compatible(dataMajor, binaryMajor); err != nil {
		return nil, err
	}

	if s.plan, err = s.makePlan(dataMajor == 0); err != nil {
		return nil, err
	}
	if dataMajor == 0 {
		log.Info("pwikit is creating its PostgreSQL data directory", "dir", cfg.Data)
		if err := s.initdb(ctx); err != nil {
			return nil, err
		}
	}
	if err := WriteConf(cfg.Data, s.plan); err != nil {
		return nil, err
	}
	if err := WriteHBA(cfg.Data, s.plan); err != nil {
		return nil, err
	}
	if s.plan.Socket != "" && s.plan.Socket != cfg.Data {
		if err := privateDir(s.plan.Socket); err != nil {
			return nil, err
		}
	}

	if err := s.launch(ctx); err != nil {
		return nil, err
	}
	if err := s.waitReady(ctx); err != nil {
		return nil, err
	}
	if err := s.ensureDatabase(ctx); err != nil {
		return nil, err
	}
	log.Info("pwikit started its PostgreSQL", "version", binaryMajor, "port", s.plan.Port, "socket", s.plan.Socket)
	return s, nil
}

func Attach(ctx context.Context, cfg Config) (string, error) {
	st, ok := readPostmaster(cfg.Data)
	if !ok || st.status != "ready" {
		return "", errors.New("pwikit serve holds the bundled PostgreSQL but it is not ready yet. Try again in a moment")
	}
	port, err := ReadPort(cfg.Data)
	if err != nil {
		return "", err
	}
	account, err := accountName()
	if err != nil {
		return "", err
	}
	goos := runtime.GOOS
	password := ""
	if goos == "windows" {
		raw, err := os.ReadFile(filepath.Join(cfg.Secrets, passwordFile))
		if err != nil {
			return "", fmt.Errorf("read the bundled PostgreSQL password: %w", err)
		}
		password = strings.TrimSpace(string(raw))
	}
	return PlanFor(goos, cfg.layout(), port, RoleFor(goos, account), password).DSN(Database), nil
}

func (s *Server) DSN() string {
	return s.plan.DSN(Database)
}

func (s *Server) Exited() <-chan struct{} {
	return s.exited
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.stop(ctx)
	if rerr := s.lock.Release(); err == nil {
		err = rerr
	}
	return err
}

func (s *Server) abandon() {
	if s.proc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		s.stop(ctx)
		cancel()
	}
	s.lock.Release()
}

func (s *Server) ExitError() error {
	tail := logTail(filepath.Join(s.cfg.Logs, startLogFile), 8)
	msg := "PostgreSQL stopped"
	if s.err != nil {
		msg += " with " + s.err.Error()
	}
	if tail != "" {
		msg += ". The last lines it wrote:\n" + indent(tail)
	}
	return fmt.Errorf("%s\n  More in %s", msg, s.cfg.Logs)
}

func (s *Server) binaryMajor(ctx context.Context) (int, error) {
	out, err := exec.CommandContext(ctx, s.layout.Binary(runtime.GOOS, "postgres"), "--version").Output()
	if err != nil {
		return 0, fmt.Errorf("run %s --version: %w", s.layout.Binary(runtime.GOOS, "postgres"), err)
	}
	return ParseVersion(string(out))
}

func (s *Server) makePlan(fresh bool) (Plan, error) {
	goos := runtime.GOOS
	account, err := accountName()
	if err != nil {
		return Plan{}, err
	}

	port := 0
	if !fresh {
		port, _ = ReadPort(s.cfg.Data)
	}
	if port == 0 || (goos == "windows" && taken("127.0.0.1:"+strconv.Itoa(port), 200*time.Millisecond)) {
		if port, err = FreePort(); err != nil {
			return Plan{}, err
		}
	}

	password := ""
	if goos == "windows" {
		if password, err = EnsurePassword(s.cfg.Secrets); err != nil {
			return Plan{}, err
		}
	}
	p := PlanFor(goos, s.layout, port, RoleFor(goos, account), password)
	p.Logs = s.cfg.Logs
	return p, nil
}

func (s *Server) initdb(ctx context.Context) error {
	data := s.cfg.Data
	if err := emptyOrMissing(data); err != nil {
		return err
	}
	staging := data + ".initdb"
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("clear %s: %w", staging, err)
	}

	args := []string{"-D", staging, "-U", s.plan.User, "-E", "UTF8", "--locale=C"}
	if s.plan.Socket != "" {
		args = append(args, "--auth-local=peer", "--auth-host=reject")
	} else {
		args = append(args, "--auth=scram-sha-256", "--pwfile="+filepath.Join(s.cfg.Secrets, passwordFile))
	}
	if err := s.tool(ctx, "initdb", args...); err != nil {
		os.RemoveAll(staging)
		return err
	}
	if err := os.Remove(data); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace the empty %s: %w", data, err)
	}
	if err := os.Rename(staging, data); err != nil {
		return fmt.Errorf("move the new data directory into %s: %w", data, err)
	}
	return nil
}

// Never a pipe. PostgreSQL inherits it and the pipe never reaches end of file.
func (s *Server) tool(ctx context.Context, name string, args ...string) error {
	logName := filepath.Join(s.cfg.Logs, toolLogFile)
	out, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	fmt.Fprintf(out, "\n--- %s %s\n", time.Now().Format(time.RFC3339), name)

	cmd := exec.CommandContext(ctx, s.layout.Binary(runtime.GOOS, name), args...)
	cmd.Stdout, cmd.Stderr = out, out
	cmd.Env = toolEnv()
	detach(cmd)
	if err := cmd.Run(); err != nil {
		tail := logTail(logName, 6)
		return fmt.Errorf("%s failed: %w\n%s", name, err, indent(tail))
	}
	return nil
}

func (s *Server) waitReady(ctx context.Context) error {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	began := time.Now()
	noted := began
	for {
		if st, ok := readPostmaster(s.cfg.Data); ok && st.pid == s.proc.pid() && st.status == "ready" {
			return nil
		}
		select {
		case <-s.exited:
			return s.ExitError()
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
		if time.Since(noted) >= progressEvery {
			noted = time.Now()
			s.cfg.logger().Info("PostgreSQL is still starting; after an unclean stop it replays its journal first",
				"waited", time.Since(began).Round(time.Second).String())
		}
	}
}

func (s *Server) ensureDatabase(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, s.plan.DSN("postgres"))
	if err != nil {
		return fmt.Errorf("connect to the bundled PostgreSQL: %w", err)
	}
	defer conn.Close(context.WithoutCancel(ctx))
	return db.EnsureDatabase(ctx, conn, Database)
}

// After a reboot the pid in a leftover postmaster.pid may belong to an unrelated program.
func (s *Server) clearLeftover(ctx context.Context) error {
	st, ok := readPostmaster(s.cfg.Data)
	if !ok {
		return nil
	}
	alive, image := processImage(st.pid)
	if !alive {
		return nil
	}
	if image == "" {
		return nil
	}
	if !isPostgres(image) {
		if err := os.Remove(filepath.Join(s.cfg.Data, postmasterFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove the stale %s: %w", postmasterFile, err)
		}
		return nil
	}
	if !samePath(image, s.layout.Binary(runtime.GOOS, "postgres")) {
		return fmt.Errorf("a PostgreSQL from %s is running on %s as process %d. Stop it before starting pwikit", image, s.cfg.Data, st.pid)
	}
	s.cfg.logger().Info("pwikit is stopping the PostgreSQL an earlier run left behind", "pid", st.pid)
	return stopLeftover(ctx, s, st.pid)
}

type postmaster struct {
	pid    int
	status string
}

func readPostmaster(data string) (postmaster, bool) {
	raw, err := os.ReadFile(filepath.Join(data, postmasterFile))
	if err != nil {
		return postmaster{}, false
	}
	lines := strings.Split(string(raw), "\n")
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil || pid <= 0 {
		return postmaster{}, false
	}
	st := postmaster{pid: pid}
	if len(lines) >= 8 {
		st.status = strings.TrimSpace(lines[7])
	}
	return st, true
}

func isPostgres(image string) bool {
	base := strings.ToLower(filepath.Base(image))
	return base == "postgres" || base == "postgres.exe"
}

func samePath(a, b string) bool {
	ai, aerr := os.Stat(a)
	bi, berr := os.Stat(b)
	if aerr == nil && berr == nil {
		return os.SameFile(ai, bi)
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func emptyOrMissing(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s holds files but no PostgreSQL data. Move them somewhere else and start pwikit again", dir)
	}
	return nil
}

func accountName() (string, error) {
	if u, err := user.Current(); err == nil && u.Username != "" {
		name := u.Username
		if i := strings.LastIndex(name, `\`); i >= 0 {
			name = name[i+1:]
		}
		return name, nil
	}
	for _, key := range []string{"USER", "USERNAME", "LOGNAME"} {
		if v := os.Getenv(key); v != "" {
			return v, nil
		}
	}
	return "", errors.New("cannot tell which account pwikit runs as, which the bundled PostgreSQL needs to name its user")
}

func toolEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(kv), "PG") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "LC_ALL=C")
}

func logTail(name string, lines int) string {
	raw, err := os.ReadFile(name)
	if err != nil {
		return ""
	}
	all := strings.Split(strings.TrimRight(string(raw), "\r\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n")
}

func indent(text string) string {
	if text == "" {
		return ""
	}
	return "    " + strings.ReplaceAll(strings.ReplaceAll(text, "\r", ""), "\n", "\n    ")
}
