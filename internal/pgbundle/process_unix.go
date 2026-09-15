//go:build unix

package pgbundle

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const systemAccount = "pwikit"

type process struct {
	cmd *exec.Cmd
}

func (p *process) pid() int {
	return p.cmd.Process.Pid
}

func (s *Server) launch(ctx context.Context) error {
	out, err := os.OpenFile(filepath.Join(s.cfg.Logs, startLogFile), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	cmd := exec.Command(s.layout.Binary(runtime.GOOS, "postgres"), "-D", s.cfg.Data)
	cmd.Stdout, cmd.Stderr = out, out
	cmd.Env = toolEnv()
	detach(cmd)
	runAs(cmd, s.as, s.cfg.Data)
	setDeathSignal(cmd.SysProcAttr)

	started := make(chan error, 1)
	go func() {
		// Pdeathsig fires when the starting thread exits, so the thread lives as long as the child.
		runtime.LockOSThread()
		err := cmd.Start()
		out.Close()
		started <- err
		if err != nil {
			return
		}
		s.err = cmd.Wait()
		close(s.exited)
	}()
	if err := <-started; err != nil {
		return fmt.Errorf("start PostgreSQL: %w", err)
	}
	s.proc = &process{cmd: cmd}
	return nil
}

func (s *Server) stop(ctx context.Context) error {
	if s.proc == nil {
		return nil
	}
	select {
	case <-s.exited:
		return nil
	default:
	}
	s.proc.cmd.Process.Signal(syscall.SIGINT)
	select {
	case <-s.exited:
		return nil
	case <-ctx.Done():
	}
	s.proc.cmd.Process.Signal(syscall.SIGQUIT)
	select {
	case <-s.exited:
	case <-time.After(10 * time.Second):
		s.proc.cmd.Process.Kill()
		<-s.exited
	}
	return errors.New("PostgreSQL did not stop in time and was stopped immediately; it replays its journal on the next start")
}

func stopLeftover(ctx context.Context, s *Server, pid int) error {
	syscall.Kill(pid, syscall.SIGINT)
	if waitGone(ctx, pid) {
		return nil
	}
	syscall.Kill(pid, syscall.SIGQUIT)
	grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if waitGone(grace, pid) {
		return nil
	}
	return fmt.Errorf("the PostgreSQL an earlier run left behind as process %d did not stop", pid)
}

func waitGone(ctx context.Context, pid int) bool {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	limit := time.After(2 * time.Minute)
	for {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-limit:
			return false
		case <-tick.C:
		}
	}
}

func detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// PostgreSQL refuses to run as root, so root hands it to the account that owns
// the data, or to an account made for it.
func postgresAccount(cfg Config) (*account, error) {
	if os.Geteuid() != 0 {
		return nil, nil
	}
	if runtime.GOOS != "linux" {
		return nil, errors.New("the bundled PostgreSQL will not run as root.\n" +
			"  Start pwikit from an ordinary account, or install it as a service with\n" +
			"  sudo pwikit service install, which runs it as the account you used sudo from")
	}
	for _, dir := range []string{cfg.Data, cfg.Root} {
		if a := ownerOf(dir); a != nil {
			return a, nil
		}
	}
	if _, err := user.Lookup(systemAccount); err != nil {
		if err := createAccount(systemAccount, cfg.Root); err != nil {
			return nil, err
		}
	}
	u, err := user.Lookup(systemAccount)
	if err != nil {
		return nil, fmt.Errorf("look up the account %s: %w", systemAccount, err)
	}
	return accountOf(u)
}

func peerRole(cfg Config) (string, error) {
	if os.Geteuid() == 0 {
		if a := ownerOf(cfg.Data); a != nil {
			return a.name, nil
		}
	}
	return accountName()
}

func ownerOf(dir string) *account {
	info, err := os.Stat(dir)
	if err != nil {
		return nil
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid == 0 {
		return nil
	}
	u, err := user.LookupId(strconv.FormatUint(uint64(st.Uid), 10))
	if err != nil {
		return nil
	}
	a, err := accountOf(u)
	if err != nil {
		return nil
	}
	return a
}

func accountOf(u *user.User) (*account, error) {
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, fmt.Errorf("account %s has uid %q: %w", u.Username, u.Uid, err)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil, fmt.Errorf("account %s has gid %q: %w", u.Username, u.Gid, err)
	}
	return &account{name: u.Username, uid: uid, gid: gid}, nil
}

func createAccount(name, home string) error {
	shell := "/bin/false"
	for _, candidate := range []string{"/usr/sbin/nologin", "/sbin/nologin"} {
		if _, err := os.Stat(candidate); err == nil {
			shell = candidate
			break
		}
	}
	var cmd *exec.Cmd
	switch {
	case lookPath("useradd"):
		cmd = exec.Command("useradd", "--system", "--user-group", "--home-dir", home, "--no-create-home", "--shell", shell, name)
	case lookPath("adduser"):
		cmd = exec.Command("adduser", "-S", "-D", "-H", "-h", home, "-s", shell, name)
	default:
		return fmt.Errorf("pwikit runs as root, and the bundled PostgreSQL needs an ordinary account to run under.\n"+
			"  Neither useradd nor adduser is here to make one. Create an account named %s, or start pwikit from an ordinary account", name)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create the account %s for the bundled PostgreSQL: %w\n%s", name, err, indent(strings.TrimSpace(string(out))))
	}
	return nil
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runAs(cmd *exec.Cmd, a *account, dir string) {
	if a == nil {
		return
	}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(a.uid), Gid: uint32(a.gid), Groups: []uint32{}}
	cmd.Dir = dir
}

func handOver(a *account, dir string) error {
	if a == nil {
		return nil
	}
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) == a.uid && int(st.Gid) == a.gid {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(path, a.uid, a.gid)
	})
}

func own(a *account, paths ...string) error {
	if a == nil {
		return nil
	}
	for _, path := range paths {
		if err := os.Lchown(path, a.uid, a.gid); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("hand %s to %s: %w", path, a.name, err)
		}
	}
	return nil
}

func reach(a *account, paths ...string) error {
	if a == nil {
		return nil
	}
	for _, path := range paths {
		for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
			info, err := os.Stat(dir)
			if err != nil {
				return err
			}
			st, ok := info.Sys().(*syscall.Stat_t)
			mode := info.Mode().Perm()
			open := !ok || mode&0o001 != 0 ||
				(int(st.Uid) == a.uid && mode&0o100 != 0) ||
				(int(st.Gid) == a.gid && mode&0o010 != 0)
			if !open {
				return fmt.Errorf("the bundled PostgreSQL runs as %s, which cannot enter %s.\n"+
					"  Move the pwikit directory somewhere every account can reach, such as /opt/pwikit or /srv/pwikit", a.name, dir)
			}
			if dir == filepath.Dir(dir) {
				break
			}
		}
	}
	return nil
}

func privateDir(dir string, a *account) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	if err := own(a, dir); err != nil {
		return err
	}
	want := os.Geteuid()
	if a != nil {
		want = a.uid
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !ok || int(st.Uid) != want {
		return fmt.Errorf("%s belongs to another account, so the PostgreSQL socket cannot be placed there", dir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return os.Chmod(dir, 0o700)
	}
	return nil
}
