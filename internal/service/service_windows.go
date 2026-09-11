package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const EventSource = "pwikit"

// A virtual account rather than LocalSystem, so the site cannot touch the whole machine.
func account(name string) string {
	return `NT SERVICE\` + name
}

func Preview(s Spec) (string, error) {
	return fmt.Sprintf("service %s, started at boot as %s, let through Windows Firewall as %q\n%s\n",
		s.Name, account(s.Name), ruleName(s.Name), windows.ComposeCommandLine(s.CommandLine())), nil
}

func connect() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return nil, errors.New("managing services needs an administrator. Open a terminal with Run as administrator and run this again")
	}
	return m, err
}

func open(name string) (*mgr.Mgr, *mgr.Service, error) {
	m, err := connect()
	if err != nil {
		return nil, nil, err
	}
	s, err := m.OpenService(name)
	if err != nil {
		m.Disconnect()
		return nil, nil, fmt.Errorf("no service named %s is installed", name)
	}
	return m, s, nil
}

func Install(spec Spec) (err error) {
	if err := spec.Validate(); err != nil {
		return err
	}
	m, err := connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	if existing, err := m.OpenService(spec.Name); err == nil {
		existing.Close()
		return fmt.Errorf("a service named %s is already installed. Remove it with pwikit service uninstall, or pick another -name", spec.Name)
	}

	s, err := m.CreateService(spec.Name, spec.Executable, mgr.Config{
		DisplayName:      DisplayName(spec.Name),
		Description:      "Serves the wikis kept in " + spec.Root,
		StartType:        mgr.StartAutomatic,
		DelayedAutoStart: true,
		ServiceStartName: account(spec.Name),
	}, spec.Args...)
	if err != nil {
		return fmt.Errorf("create service %s: %w", spec.Name, err)
	}
	defer s.Close()
	defer func() {
		if err != nil {
			s.Delete()
		}
	}()

	actions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: time.Minute},
	}
	if err := s.SetRecoveryActions(actions, uint32((24 * time.Hour).Seconds())); err != nil {
		return err
	}
	if err := s.SetRecoveryActionsOnNonCrashFailures(true); err != nil {
		return err
	}
	eventlog.InstallAsEventCreate(EventSource, eventlog.Error|eventlog.Warning|eventlog.Info)

	if err := grant(spec.Root, account(spec.Name)); err != nil {
		return err
	}
	if err := openFirewall(spec.Name, spec.Executable); err != nil {
		fmt.Fprintf(os.Stderr, "could not let pwikit through Windows Firewall: %v; visitors from other machines stay blocked until it is allowed in\n", err)
	}
	if err := s.Start(); err != nil {
		return fmt.Errorf("start service %s: %w", spec.Name, err)
	}
	return nil
}

func grant(dir, who string) error {
	cmd := exec.Command("icacls", dir, "/grant", who+":(OI)(CI)M", "/C", "/Q")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("give %s access to %s: %w", who, dir, err)
	}
	return nil
}

func Uninstall(name string) error {
	m, s, err := open(name)
	if err != nil {
		return err
	}
	defer m.Disconnect()
	defer s.Close()
	if err := stopAndWait(s); err != nil {
		return err
	}
	if err := s.Delete(); err != nil {
		return err
	}
	closeFirewall(name)
	return nil
}

func ruleName(name string) string {
	return DisplayName(name)
}

func openFirewall(name, exe string) error {
	closeFirewall(name)
	return netsh(`advfirewall firewall add rule name="` + ruleName(name) + `" dir=in action=allow program="` + exe + `" enable=yes profile=any`)
}

func closeFirewall(name string) error {
	return netsh(`advfirewall firewall delete rule name="` + ruleName(name) + `"`)
}

// netsh needs the quote after the equals sign, which normal argument quoting cannot produce.
func netsh(args string) error {
	cmd := exec.Command("netsh")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "netsh " + args}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %s: %w: %s", args, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Start(name string) error {
	m, s, err := open(name)
	if err != nil {
		return err
	}
	defer m.Disconnect()
	defer s.Close()
	return s.Start()
}

func Stop(name string) error {
	m, s, err := open(name)
	if err != nil {
		return err
	}
	defer m.Disconnect()
	defer s.Close()
	return stopAndWait(s)
}

// Read access only, since mgr asks for full control and that needs an administrator.
func Status(name string) error {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return fmt.Errorf("reach the service manager: %w", err)
	}
	defer windows.CloseServiceHandle(h)
	target, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	sh, err := windows.OpenService(h, target, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return fmt.Errorf("no service named %s is installed", name)
	}
	s := &mgr.Service{Name: name, Handle: sh}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return err
	}
	fmt.Printf("%s is %s\n", name, stateName(st.State))
	return nil
}

func stopAndWait(s *mgr.Service) error {
	st, err := s.Query()
	if err != nil {
		return err
	}
	if st.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err != nil && st.State != svc.StopPending {
			return fmt.Errorf("stop service: %w", err)
		}
	}
	deadline := time.Now().Add(StopTimeout * time.Second)
	for time.Now().Before(deadline) {
		if st, err = s.Query(); err != nil {
			return err
		}
		if st.State == svc.Stopped {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("the service did not stop in time")
}

func stateName(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "starting"
	case svc.StopPending:
		return "stopping"
	case svc.Running:
		return "running"
	case svc.PausePending, svc.Paused, svc.ContinuePending:
		return "paused"
	}
	return "in an unknown state"
}
