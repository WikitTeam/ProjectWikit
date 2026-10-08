//go:build unix

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/update"
)

// The bundled PostgreSQL lets in only the account that owns the data
// directory, so root hands every step that touches the database to that account.
func ownerCredential(root string) (*syscall.SysProcAttr, []string) {
	if os.Geteuid() != 0 {
		return nil, nil
	}
	uid, gid, ok := update.Account(root)
	if !ok || uid == 0 {
		return nil, nil
	}
	env := os.Environ()
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
		env = append(env, "HOME="+u.HomeDir, "USER="+u.Username, "LOGNAME="+u.Username)
	}
	return &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid}}, env
}

func nameless(root string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	uid, _, ok := update.Account(root)
	if !ok || uid == 0 {
		return nil
	}
	if _, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err != nil {
		return fmt.Errorf("%s belongs to uid %d, which has no account on this machine. "+
			"Create an account with that uid, or hand the directory to an existing account with chown -R", root, uid)
	}
	return nil
}

func runOwner(ctx context.Context, root, exe string, args ...string) ([]byte, error) {
	if err := nameless(root); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = root
	cmd.SysProcAttr, cmd.Env = ownerCredential(root)
	if cmd.SysProcAttr != nil {
		done := handConfig(cmd, root)
		defer done()
	}
	return cmd.CombinedOutput()
}

func handConfig(cmd *exec.Cmd, root string) func() {
	p, err := paths.New(root)
	if err != nil {
		return func() {}
	}
	var closers []func()
	if data, err := os.ReadFile(p.Config()); err == nil {
		closers = append(closers, handOn(cmd, config.FDEnv, data))
	}
	if name := os.Getenv(envDatabasePasswordFile); name != "" {
		if data, err := os.ReadFile(name); err == nil {
			closers = append(closers, handOn(cmd, envDatabasePasswordFD, data))
		}
	}
	return func() {
		for _, c := range closers {
			c()
		}
	}
}

func handOn(cmd *exec.Cmd, env string, data []byte) func() {
	r, w, err := os.Pipe()
	if err != nil {
		return func() {}
	}
	go func() {
		w.Write(data)
		w.Close()
	}()
	cmd.ExtraFiles = append(cmd.ExtraFiles, r)
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", env, 2+len(cmd.ExtraFiles)))
	return func() { r.Close() }
}

func asOwner(root string) (bool, error) {
	if err := nameless(root); err != nil {
		return true, err
	}
	attr, env := ownerCredential(root)
	if attr == nil {
		return false, nil
	}
	exe, err := runningExecutable()
	if err != nil {
		return true, err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Dir = root
	cmd.SysProcAttr, cmd.Env = attr, env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	done := handConfig(cmd, root)
	err = cmd.Run()
	done()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
	return true, err
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
