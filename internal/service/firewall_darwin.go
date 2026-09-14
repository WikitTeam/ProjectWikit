package service

import (
	"html"
	"os"
	"os/exec"
	"strings"
)

const socketfilterfw = "/usr/libexec/ApplicationFirewall/socketfilterfw"

func allowProgram(exe string) {
	if !asDaemon() || !firewallOn() {
		return
	}
	runTool(socketfilterfw, "--add", exe)
	runTool(socketfilterfw, "--unblockapp", exe)
}

func removeProgram(exe string) {
	if !asDaemon() || exe == "" || !firewallOn() {
		return
	}
	runTool(socketfilterfw, "--remove", exe)
}

func firewallOn() bool {
	out, err := exec.Command(socketfilterfw, "--getglobalstate").Output()
	return err == nil && strings.Contains(string(out), "enabled")
}

func programIn(plistPath string) string {
	raw, err := os.ReadFile(plistPath)
	if err != nil {
		return ""
	}
	_, rest, ok := strings.Cut(string(raw), "<key>ProgramArguments</key>")
	if !ok {
		return ""
	}
	_, rest, ok = strings.Cut(rest, "<string>")
	if !ok {
		return ""
	}
	value, _, ok := strings.Cut(rest, "</string>")
	if !ok {
		return ""
	}
	return html.UnescapeString(value)
}
