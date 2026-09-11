package service

import (
	"encoding/xml"
	"strings"
	"testing"
)

func sampleSpec() Spec {
	return Spec{
		Name:       "pwikit",
		Executable: "/opt/my wiki/pwikit",
		Root:       "/opt/my wiki",
		User:       "alice",
		LogFile:    "/opt/my wiki/logs/pwikit.log",
		CrashFile:  "/opt/my wiki/logs/pwikit-stderr.log",
		Args:       []string{"serve", "-data-dir", "/opt/my wiki", "-acme-email", "a&b<c>@example.com"},
	}
}

func lineWith(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func TestValidName(t *testing.T) {
	for _, name := range []string{"pwikit", "wiki-2", "a.b_c"} {
		if err := ValidName(name); err != nil {
			t.Errorf("ValidName(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"", "-x", "has space", "a/b", `a\b`, "wiki$", strings.Repeat("a", 65)} {
		if err := ValidName(name); err == nil {
			t.Errorf("ValidName(%q) = nil, want an error", name)
		}
	}
}

func TestDisplayName(t *testing.T) {
	cases := map[string]string{DefaultName: "ProjectWikit", "wiki-2": "ProjectWikit wiki-2"}
	for name, want := range cases {
		if got := DisplayName(name); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSystemdQuotesEveryArgument(t *testing.T) {
	got := lineWith(sampleSpec().Systemd(), "ExecStart=")
	want := `ExecStart="/opt/my wiki/pwikit" "serve" "-data-dir" "/opt/my wiki" "-acme-email" "a&b<c>@example.com"`
	if got != want {
		t.Errorf("ExecStart line = %q, want %q", got, want)
	}
}

func TestSystemdEscapesSpecifiersVariablesAndQuotes(t *testing.T) {
	s := sampleSpec()
	s.Args = []string{"serve", `-secret-key`, `50%$HOME"\`}
	got := lineWith(s.Systemd(), "ExecStart=")
	want := `ExecStart="/opt/my wiki/pwikit" "serve" "-secret-key" "50%%$$HOME\"\\"`
	if got != want {
		t.Errorf("ExecStart line = %q, want %q", got, want)
	}
}

func TestSystemdWorkingDirectoryKeepsDollars(t *testing.T) {
	s := sampleSpec()
	s.Root = "/srv/100%/$wiki"
	if got, want := lineWith(s.Systemd(), "WorkingDirectory="), "WorkingDirectory=/srv/100%%/$wiki"; got != want {
		t.Errorf("WorkingDirectory line = %q, want %q", got, want)
	}
}

func TestSystemdStopsOnlyTheMainProcess(t *testing.T) {
	unit := sampleSpec().Systemd()
	for _, want := range []string{"KillMode=mixed", "User=alice", "Restart=on-failure", "WantedBy=multi-user.target", "AmbientCapabilities=CAP_NET_BIND_SERVICE"} {
		if !strings.Contains(unit, want+"\n") {
			t.Errorf("Systemd() lacks %q", want)
		}
	}
}

func TestSystemdRecordsTheOpenedPorts(t *testing.T) {
	s := sampleSpec()
	s.Opened = Firewall{Tool: "firewalld", Ports: []int{80, 443}}
	got := ParseFirewall(s.Systemd())
	if got.Tool != "firewalld" || len(got.Ports) != 2 || got.Ports[0] != 80 || got.Ports[1] != 443 {
		t.Errorf("ParseFirewall(Systemd()) = %+v, want firewalld [80 443]", got)
	}
}

func TestSystemdWithoutOpenedPortsRecordsNothing(t *testing.T) {
	unit := sampleSpec().Systemd()
	if got := ParseFirewall(unit); got.Tool != "" || got.Ports != nil {
		t.Errorf("ParseFirewall(Systemd()) = %+v, want nothing", got)
	}
	if strings.Contains(unit, firewallMarker) {
		t.Errorf("Systemd() holds %q with nothing opened", firewallMarker)
	}
}

func TestParseFirewallIgnoresGarbage(t *testing.T) {
	for _, unit := range []string{"", "# pwikit-firewall\n", "# pwikit-firewall ufw\n", "[Unit]\nDescription=x\n"} {
		if got := ParseFirewall(unit); got.Tool != "" || got.Ports != nil {
			t.Errorf("ParseFirewall(%q) = %+v, want nothing", unit, got)
		}
	}
}

func TestSystemdOmitsUserWhenUnset(t *testing.T) {
	s := sampleSpec()
	s.User = ""
	if got := lineWith(s.Systemd(), "User="); got != "" {
		t.Errorf("User line = %q, want none", got)
	}
}

type plistDoc struct {
	Dict struct {
		Keys    []string `xml:"key"`
		Strings []string `xml:"string"`
		Arrays  []struct {
			Strings []string `xml:"string"`
		} `xml:"array"`
	} `xml:"dict"`
}

func parsePlist(t *testing.T, text string) plistDoc {
	t.Helper()
	var doc plistDoc
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.Strict = false
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("xml.Decode(Launchd()) error = %v", err)
	}
	return doc
}

func TestLaunchdArgumentsSurviveEscaping(t *testing.T) {
	s := sampleSpec()
	doc := parsePlist(t, s.Launchd(true))
	if len(doc.Dict.Arrays) != 1 {
		t.Fatalf("Launchd() arrays = %d, want 1", len(doc.Dict.Arrays))
	}
	got := doc.Dict.Arrays[0].Strings
	want := s.CommandLine()
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("ProgramArguments = %q, want %q", got, want)
	}
}

func TestLaunchdDaemonNamesTheUser(t *testing.T) {
	s := sampleSpec()
	if doc := parsePlist(t, s.Launchd(true)); !contains(doc.Dict.Keys, "UserName") {
		t.Errorf("Launchd(true) keys = %q, want UserName among them", doc.Dict.Keys)
	}
	if doc := parsePlist(t, s.Launchd(false)); contains(doc.Dict.Keys, "UserName") {
		t.Errorf("Launchd(false) keys = %q, want no UserName", doc.Dict.Keys)
	}
}

func TestLaunchdRestartsOnlyAfterAFailure(t *testing.T) {
	plist := sampleSpec().Launchd(true)
	want := "<key>KeepAlive</key>\n\t<dict>\n\t\t<key>SuccessfulExit</key>\n\t\t<false/>"
	if !strings.Contains(plist, want) {
		t.Errorf("Launchd() lacks %q", want)
	}
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
