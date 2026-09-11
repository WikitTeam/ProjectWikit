//go:build unix

package pgbundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

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

func refuseSuperuser() error {
	if os.Geteuid() != 0 {
		return nil
	}
	return errors.New("the bundled PostgreSQL will not run as root.\n" +
		"  Start pwikit from an ordinary account, or install it as a service with\n" +
		"  sudo pwikit service install, which runs it as the account you used sudo from")
}

func privateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !ok || int(st.Uid) != os.Geteuid() {
		return fmt.Errorf("%s belongs to another account, so the PostgreSQL socket cannot be placed there", dir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return os.Chmod(dir, 0o700)
	}
	return nil
}
