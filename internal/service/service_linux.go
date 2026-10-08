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
	if len(s.UpdateArgs) > 0 {
		unit, timer := s.SystemdUpdate()
		text += "\n# " + filepath.Join(unitDir, UpdateName(s.Name)+".service") + "\n" + unit
		text += "\n# " + filepath.Join(unitDir, UpdateName(s.Name)+".timer") + "\n" + timer
	}
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
	path := unitPath(s.Name)
	if _, err := os.Stat(path); err == nil {
		in, _ := Lookup(s.Name)
		if in.DataDir != s.Root {
			return fmt.Errorf("a service named %s is already installed. Remove it with pwikit service uninstall, or pick another -name", s.Name)
		}
		if err := runTool("systemctl", "stop", s.Name+".service"); err != nil {
			return err
		}
	}
	handState(s.State, u)
	if err := checkAccess(s.Root, u); err != nil {
		return err
	}
	s.Opened = openPorts(s.Ports)
	if err := os.WriteFile(path, []byte(s.Systemd()), 0o644); err != nil {
		closePorts(s.Opened)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if len(s.UpdateArgs) > 0 {
		unit, timer := s.SystemdUpdate()
		if err := os.WriteFile(filepath.Join(unitDir, UpdateName(s.Name)+".service"), []byte(unit), 0o644); err != nil {
			return fmt.Errorf("write the update unit: %w", err)
		}
		if err := os.WriteFile(filepath.Join(unitDir, UpdateName(s.Name)+".timer"), []byte(timer), 0o644); err != nil {
			return fmt.Errorf("write the update timer: %w", err)
		}
	}
	if err := runTool("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := runTool("systemctl", "enable", "--now", s.Name+".service"); err != nil {
		return err
	}
	if len(s.UpdateArgs) > 0 {
		return runTool("systemctl", "enable", "--now", UpdateName(s.Name)+".timer")
	}
	return nil
}

func Running(name string) (bool, error) {
	err := exec.Command("systemctl", "is-active", "--quiet", name+".service").Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return false, nil
	}
	return err == nil, err
}

func removeUpdateTask(name string) {
	timer := filepath.Join(unitDir, UpdateName(name)+".timer")
	if _, err := os.Stat(timer); err != nil {
		return
	}
	runTool("systemctl", "disable", "--now", UpdateName(name)+".timer")
	os.Remove(timer)
	os.Remove(filepath.Join(unitDir, UpdateName(name)+".service"))
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
	removeUpdateTask(name)
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

func Lookup(name string) (Installed, bool) {
	unit, err := os.ReadFile(unitPath(name))
	if err != nil {
		return Installed{}, false
	}
	args := systemdArgs(string(unit))
	in := Installed{Executable: systemdProgram(string(unit)), DataDir: dataDirArg(args), ServeArgs: serveExtra(args)}
	in.User, _ = systemdValue(string(unit), "User")
	if update, err := os.ReadFile(filepath.Join(unitDir, UpdateName(name)+".service")); err == nil {
		in.UpdateExecutable = systemdProgram(string(update))
	}
	return in, true
}

func PointUpdateAt(name, exe string) (bool, error) {
	return pointUnitAt(filepath.Join(unitDir, UpdateName(name)+".service"), exe)
}

func PointServiceAt(name, exe string) (bool, error) {
	return pointUnitAt(unitPath(name), exe)
}

func pointUnitAt(path, exe string) (bool, error) {
	unit, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	next, changed := pointSystemdAt(string(unit), exe)
	if !changed {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return false, err
	}
	return true, exec.Command("systemctl", "daemon-reload").Run()
}
