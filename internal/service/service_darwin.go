package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func asDaemon() bool {
	return os.Geteuid() == 0
}

func plistPath(name string) (string, error) {
	if asDaemon() {
		return filepath.Join("/Library/LaunchDaemons", name+".plist"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", name+".plist"), nil
}

func domain() string {
	if asDaemon() {
		return "system"
	}
	return "gui/" + strconv.Itoa(os.Getuid())
}

func Preview(s Spec) (string, error) {
	if u, err := serviceUser(s.User); err == nil {
		s.User = u.Username
	}
	path, err := plistPath(s.Name)
	if err != nil {
		return "", err
	}
	return "<!-- " + path + " -->\n" + s.Launchd(asDaemon()), nil
}

func Install(s Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	u, err := serviceUser(s.User)
	if err != nil {
		return err
	}
	s.User = u.Username
	if err := checkAccess(s.Root, u); err != nil {
		return err
	}
	path, err := plistPath(s.Name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("a service named %s is already installed. Remove it with pwikit service uninstall, or pick another -name", s.Name)
	}

	if s.CrashFile != "" {
		logs := filepath.Dir(s.CrashFile)
		if err := os.MkdirAll(logs, 0o755); err != nil {
			return err
		}
		if asDaemon() {
			uid, _ := strconv.Atoi(u.Uid)
			gid, _ := strconv.Atoi(u.Gid)
			if err := os.Chown(logs, uid, gid); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(s.Launchd(asDaemon())), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	allowProgram(s.Executable)
	if err := runTool("launchctl", "bootstrap", domain(), path); err != nil {
		os.Remove(path)
		if !asDaemon() {
			return fmt.Errorf("%w\n  A Mac nobody is logged in to has no session to start an agent in. Install with sudo so it starts at boot instead", err)
		}
		return err
	}
	return nil
}

func Uninstall(name string) error {
	path, err := plistPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("no service named %s is installed%s", name, elsewhere())
	}
	exe := programIn(path)
	runTool("launchctl", "bootout", domain()+"/"+name)
	if err := os.Remove(path); err != nil {
		return err
	}
	removeProgram(exe)
	return nil
}

func Start(name string) error {
	return runTool("launchctl", "kickstart", domain()+"/"+name)
}

func Stop(name string) error {
	return runTool("launchctl", "kill", "SIGTERM", domain()+"/"+name)
}

func Status(name string) error {
	err := runTool("launchctl", "print", domain()+"/"+name)
	if err != nil {
		return errors.New("no service named " + name + " is loaded" + elsewhere())
	}
	return nil
}

func elsewhere() string {
	if asDaemon() {
		return ". One installed without sudo is only visible without sudo"
	}
	return ". One installed with sudo is only visible with sudo"
}
