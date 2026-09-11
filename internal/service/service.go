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

	Ports  []int
	Opened Firewall
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
