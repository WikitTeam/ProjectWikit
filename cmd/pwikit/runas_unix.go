//go:build unix

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/WikitTeam/ProjectWikit/internal/update"
)

// The bundled PostgreSQL lets in only the account that owns the data
// directory, so root hands every step that touches the database to that account.
func ownerCredential(root string) (*syscall.SysProcAttr, []string) {
	if os.Geteuid() != 0 {
		return nil, nil
	}
	uid, gid, ok := update.Owner(root)
	if !ok || uid == 0 {
		return nil, nil
	}
	env := os.Environ()
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
		env = append(env, "HOME="+u.HomeDir, "USER="+u.Username, "LOGNAME="+u.Username)
	}
	return &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid}}, env
}

func runOwner(ctx context.Context, root, exe string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = root
	cmd.SysProcAttr, cmd.Env = ownerCredential(root)
	return cmd.CombinedOutput()
}

func asOwner(root string) (bool, error) {
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
	err = cmd.Run()
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
