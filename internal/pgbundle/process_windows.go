//go:build windows

package pgbundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const stillActive = 259

type process struct {
	id int
}

func (p *process) pid() int {
	return p.id
}

// postgres.exe refuses an administrator's token and only pg_ctl knows how to drop it.
func (s *Server) launch(ctx context.Context) error {
	startLog := filepath.Join(s.cfg.Logs, startLogFile)
	if err := os.Remove(startLog); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- s.tool(ctx, "pg_ctl", "start", "-D", s.cfg.Data, "-l", startLog, "-w", "-t", "86400", "-s")
	}()
	began := time.Now()
	for waiting := true; waiting; {
		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("start PostgreSQL: %w\n%s", err, indent(logTail(startLog, 8)))
			}
			waiting = false
		case <-time.After(progressEvery):
			s.cfg.logger().Info("PostgreSQL is still starting; after an unclean stop it replays its journal first",
				"waited", time.Since(began).Round(time.Second).String())
		}
	}

	st, ok := readPostmaster(s.cfg.Data)
	if !ok {
		return errors.New("PostgreSQL reported it started but left no " + postmasterFile)
	}
	p, err := os.FindProcess(st.pid)
	if err != nil {
		return fmt.Errorf("watch PostgreSQL process %d: %w", st.pid, err)
	}
	s.proc = &process{id: st.pid}
	go func() {
		_, s.err = p.Wait()
		close(s.exited)
	}()
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
	if err := s.pgCtlStop(ctx, "fast"); err == nil {
		select {
		case <-s.exited:
			return nil
		case <-time.After(10 * time.Second):
		}
	}
	grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	s.pgCtlStop(grace, "immediate")
	select {
	case <-s.exited:
	case <-grace.Done():
	}
	return errors.New("PostgreSQL did not stop in time and was stopped immediately; it replays its journal on the next start")
}

func (s *Server) pgCtlStop(ctx context.Context, mode string) error {
	return s.tool(ctx, "pg_ctl", "stop", "-D", s.cfg.Data, "-m", mode, "-w", "-t", "86400", "-s")
}

func stopLeftover(ctx context.Context, s *Server, pid int) error {
	if err := s.pgCtlStop(ctx, "fast"); err == nil {
		return nil
	}
	grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if err := s.pgCtlStop(grace, "immediate"); err != nil {
		return fmt.Errorf("the PostgreSQL an earlier run left behind as process %d did not stop: %w", pid, err)
	}
	return nil
}

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}

func refuseSuperuser() error {
	return nil
}

func privateDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

func processImage(pid int) (bool, string) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return !errors.Is(err, windows.ERROR_INVALID_PARAMETER), ""
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err == nil && code != stillActive {
		return false, ""
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return true, ""
	}
	return true, windows.UTF16ToString(buf[:size])
}
