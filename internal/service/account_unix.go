//go:build linux || darwin

package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

func serviceUser(given string) (*user.User, error) {
	name := given
	if name == "" {
		name = os.Getenv("SUDO_USER")
	}
	if name == "" && os.Geteuid() != 0 {
		if u, err := user.Current(); err == nil {
			return u, nil
		}
	}
	if name == "" || name == "root" {
		return nil, errors.New("the bundled PostgreSQL will not run as root, so the service needs an ordinary account.\n" +
			"  Run the install with sudo from that account, or name it with -user")
	}
	u, err := user.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("no account named %q: %w", name, err)
	}
	return u, nil
}

func checkAccess(root string, u *user.User) error {
	uid, _ := strconv.Atoi(u.Uid)
	groups, _ := u.GroupIds()
	groups = append(groups, u.Gid)

	for _, dir := range []string{root, filepath.Join(root, "pgdata")} {
		info, err := os.Stat(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != uid {
			return fmt.Errorf("%s belongs to another account than %s, and PostgreSQL only opens data its own account owns.\n"+
				"  Hand the directory over with: sudo chown -R %s: %s", dir, u.Username, u.Username, root)
		}
	}

	for dir := filepath.Dir(root); ; dir = filepath.Dir(dir) {
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		mode := info.Mode().Perm()
		reachable := !ok || mode&0o001 != 0 ||
			(int(st.Uid) == uid && mode&0o100 != 0) ||
			(slices.Contains(groups, strconv.Itoa(int(st.Gid))) && mode&0o010 != 0)
		if !reachable {
			return fmt.Errorf("the account %s cannot reach %s, because %s is closed to it.\n"+
				"  Move the pwikit directory somewhere it can reach, such as /opt/pwikit or its home directory", u.Username, root, dir)
		}
		if dir == filepath.Dir(dir) {
			return nil
		}
	}
}

func runTool(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func needRoot(action string) error {
	if os.Geteuid() == 0 {
		return nil
	}
	return fmt.Errorf("%s a service needs root. Run the same command again with sudo in front", action)
}
