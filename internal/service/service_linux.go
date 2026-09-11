package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const unitDir = "/etc/systemd/system"

func unitPath(name string) string {
	return filepath.Join(unitDir, name+".service")
}

func Preview(s Spec) (string, error) {
	if u, err := serviceUser(s.User); err == nil {
		s.User = u.Username
	}
	text := "# " + unitPath(s.Name) + "\n" + s.Systemd()
	if len(s.Ports) > 0 {
		text += fmt.Sprintf("\n# Install also opens %v/tcp when firewalld or ufw is running and they are closed.\n", s.Ports)
	}
	return text, nil
}

func Install(s Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := needRoot("installing"); err != nil {
		return err
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return errors.New("this machine does not run systemd, so pwikit cannot register itself.\n" +
			"  Have your init system run the command that pwikit service print shows")
	}
	u, err := serviceUser(s.User)
	if err != nil {
		return err
	}
	s.User = u.Username
	if err := checkAccess(s.Root, u); err != nil {
		return err
	}
	path := unitPath(s.Name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("a service named %s is already installed. Remove it with pwikit service uninstall, or pick another -name", s.Name)
	}
	s.Opened = openPorts(s.Ports)
	if err := os.WriteFile(path, []byte(s.Systemd()), 0o644); err != nil {
		closePorts(s.Opened)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := runTool("systemctl", "daemon-reload"); err != nil {
		return err
	}
	return runTool("systemctl", "enable", "--now", s.Name+".service")
}

func Uninstall(name string) error {
	if err := needRoot("removing"); err != nil {
		return err
	}
	path := unitPath(name)
	unit, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no service named %s is installed", name)
	}
	if err := runTool("systemctl", "disable", "--now", name+".service"); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	closePorts(ParseFirewall(string(unit)))
	return runTool("systemctl", "daemon-reload")
}

func Start(name string) error {
	if err := needRoot("starting"); err != nil {
		return err
	}
	return runTool("systemctl", "start", name+".service")
}

func Stop(name string) error {
	if err := needRoot("stopping"); err != nil {
		return err
	}
	return runTool("systemctl", "stop", name+".service")
}

// systemctl exits nonzero for a stopped service, which is an answer and not an error.
func Status(name string) error {
	err := runTool("systemctl", "status", "--no-pager", name+".service")
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return nil
	}
	return err
}
