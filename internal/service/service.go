// Package service answers how pwikit starts itself when the machine boots.
package service

import (
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const DefaultName = "pwikit"

const StopTimeout = 180

type Spec struct {
	Name       string
	Executable string
	Root       string
	User       string
	LogFile    string
	CrashFile  string
	Args       []string

	UpdateArgs []string

	Ports  []int
	Opened Firewall
}

const UpdateEvery = 10 * 60

func UpdateName(name string) string {
	return name + "-update"
}

// Only ports install itself opened, so uninstall never closes one another program needs.
type Firewall struct {
	Tool  string
	Ports []int
}

const firewallMarker = "# pwikit-firewall"

func (f Firewall) line() string {
	if f.Tool == "" || len(f.Ports) == 0 {
		return ""
	}
	fields := []string{firewallMarker, f.Tool}
	for _, port := range f.Ports {
		fields = append(fields, strconv.Itoa(port))
	}
	return strings.Join(fields, " ")
}

func ParseFirewall(unit string) Firewall {
	for _, line := range strings.Split(unit, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), firewallMarker+" ")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			return Firewall{}
		}
		f := Firewall{Tool: fields[0]}
		for _, field := range fields[1:] {
			if port, err := strconv.Atoi(field); err == nil && port > 0 {
				f.Ports = append(f.Ports, port)
			}
		}
		return f
	}
	return Firewall{}
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func ValidName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("service name %q may only hold letters, digits, ., - and _, and must start with a letter or digit", name)
	}
	return nil
}

func DisplayName(name string) string {
	if name == DefaultName {
		return "ProjectWikit"
	}
	return "ProjectWikit " + name
}

func (s Spec) Validate() error {
	if err := ValidName(s.Name); err != nil {
		return err
	}
	if s.Executable == "" || s.Root == "" {
		return errors.New("a service needs the executable and the state directory")
	}
	return nil
}

func (s Spec) CommandLine() []string {
	return append([]string{s.Executable}, s.Args...)
}

func (s Spec) Systemd() string {
	var b strings.Builder
	if line := s.Opened.line(); line != "" {
		b.WriteString("# The ports below were opened by pwikit service install, and uninstall closes them again.\n")
		b.WriteString(line + "\n")
	}
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s\n", systemdEscape(DisplayName(s.Name)))
	b.WriteString("Wants=network-online.target\n")
	b.WriteString("After=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	if s.User != "" {
		fmt.Fprintf(&b, "User=%s\n", systemdEscape(s.User))
	}
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", systemdEscape(s.Root))
	quoted := make([]string, 0, len(s.Args)+1)
	for _, arg := range s.CommandLine() {
		quoted = append(quoted, systemdQuote(arg))
	}
	fmt.Fprintf(&b, "ExecStart=%s\n", strings.Join(quoted, " "))
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")
	// pwikit stops PostgreSQL itself. SIGTERM to the whole group would cut that short.
	b.WriteString("KillMode=mixed\n")
	fmt.Fprintf(&b, "TimeoutStopSec=%d\n", StopTimeout)
	b.WriteString("AmbientCapabilities=CAP_NET_BIND_SERVICE\n")
	b.WriteString("LimitNOFILE=65536\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String()
}

func (s Spec) SystemdUpdate() (service, timer string) {
	quoted := make([]string, 0, len(s.UpdateArgs)+1)
	for _, arg := range append([]string{s.Executable}, s.UpdateArgs...) {
		quoted = append(quoted, systemdQuote(arg))
	}
	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s update\n", systemdEscape(DisplayName(s.Name)))
	b.WriteString("Wants=network-online.target\n")
	b.WriteString("After=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=oneshot\n")
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", systemdEscape(s.Root))
	fmt.Fprintf(&b, "ExecStart=%s\n", strings.Join(quoted, " "))
	b.WriteString("TimeoutStartSec=4h\n")
	service = b.String()

	b.Reset()
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s update check\n\n", systemdEscape(DisplayName(s.Name)))
	b.WriteString("[Timer]\n")
	b.WriteString("OnBootSec=5min\n")
	fmt.Fprintf(&b, "OnUnitInactiveSec=%ds\n", UpdateEvery)
	b.WriteString("AccuracySec=1min\n")
	fmt.Fprintf(&b, "Unit=%s.service\n\n", UpdateName(s.Name))
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=timers.target\n")
	return service, b.String()
}

func (s Spec) LaunchdUpdate() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	plistString(&b, "Label", UpdateName(s.Name))
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range append([]string{s.Executable}, s.UpdateArgs...) {
		b.WriteString("\t\t<string>" + xmlText(arg) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	plistString(&b, "WorkingDirectory", s.Root)
	fmt.Fprintf(&b, "\t<key>StartInterval</key>\n\t<integer>%d</integer>\n", UpdateEvery)
	b.WriteString("\t<key>RunAtLoad</key>\n\t<false/>\n")
	if s.CrashFile != "" {
		updateLog := strings.TrimSuffix(s.CrashFile, "pwikit-stderr.log") + "update-stderr.log"
		plistString(&b, "StandardOutPath", updateLog)
		plistString(&b, "StandardErrorPath", updateLog)
	}
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func (s Spec) TaskXML() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-16"?>` + "\n")
	b.WriteString(`<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">` + "\n")
	b.WriteString("  <RegistrationInfo><Description>" + xmlText("Looks for and installs new releases of "+DisplayName(s.Name)) + "</Description></RegistrationInfo>\n")
	b.WriteString("  <Triggers><TimeTrigger>")
	fmt.Fprintf(&b, "<Repetition><Interval>PT%dM</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition>", UpdateEvery/60)
	b.WriteString("<StartBoundary>2026-01-01T00:00:00</StartBoundary><Enabled>true</Enabled></TimeTrigger></Triggers>\n")
	b.WriteString("  <Principals><Principal id=\"Author\"><UserId>S-1-5-18</UserId><RunLevel>HighestAvailable</RunLevel></Principal></Principals>\n")
	b.WriteString("  <Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>")
	b.WriteString("<DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries><StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>")
	b.WriteString("<StartWhenAvailable>true</StartWhenAvailable><AllowStartOnDemand>true</AllowStartOnDemand><Enabled>true</Enabled>")
	b.WriteString("<ExecutionTimeLimit>PT4H</ExecutionTimeLimit></Settings>\n")
	b.WriteString("  <Actions Context=\"Author\"><Exec>")
	b.WriteString("<Command>" + xmlText(s.Executable) + "</Command>")
	b.WriteString("<Arguments>" + xmlText(composeArgs(s.UpdateArgs)) + "</Arguments>")
	b.WriteString("<WorkingDirectory>" + xmlText(s.Root) + "</WorkingDirectory>")
	b.WriteString("</Exec></Actions>\n</Task>\n")
	return b.String()
}

func composeArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "" && !strings.ContainsAny(arg, " \t\"") {
			quoted = append(quoted, arg)
			continue
		}
		var b strings.Builder
		b.WriteByte('"')
		slashes := 0
		for _, r := range arg {
			switch r {
			case '\\':
				slashes++
			case '"':
				b.WriteString(strings.Repeat(`\`, slashes*2+1))
				b.WriteRune(r)
				slashes = 0
				continue
			default:
				if slashes > 0 {
					b.WriteString(strings.Repeat(`\`, slashes))
					slashes = 0
				}
				b.WriteRune(r)
			}
		}
		b.WriteString(strings.Repeat(`\`, slashes*2))
		b.WriteByte('"')
		quoted = append(quoted, b.String())
	}
	return strings.Join(quoted, " ")
}

func systemdEscape(value string) string {
	return strings.NewReplacer("%", "%%", "\n", " ").Replace(value)
}

func systemdQuote(arg string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", "$$").Replace(systemdEscape(arg))
	return `"` + escaped + `"`
}

func (s Spec) Launchd(daemon bool) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	plistString(&b, "Label", s.Name)
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range s.CommandLine() {
		b.WriteString("\t\t<string>" + xmlText(arg) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	plistString(&b, "WorkingDirectory", s.Root)
	if daemon && s.User != "" {
		plistString(&b, "UserName", s.User)
	}
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	b.WriteString("\t<key>KeepAlive</key>\n\t<dict>\n\t\t<key>SuccessfulExit</key>\n\t\t<false/>\n\t</dict>\n")
	b.WriteString("\t<key>ThrottleInterval</key>\n\t<integer>10</integer>\n")
	fmt.Fprintf(&b, "\t<key>ExitTimeOut</key>\n\t<integer>%d</integer>\n", StopTimeout)
	if s.CrashFile != "" {
		plistString(&b, "StandardOutPath", s.CrashFile)
		plistString(&b, "StandardErrorPath", s.CrashFile)
	}
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func plistString(b *strings.Builder, key, value string) {
	b.WriteString("\t<key>" + key + "</key>\n\t<string>" + xmlText(value) + "</string>\n")
}

func xmlText(value string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(value))
	return b.String()
}
