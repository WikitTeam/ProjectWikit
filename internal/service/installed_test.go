package service

import (
	"strings"
	"testing"
)

func TestSystemdFirstReadsWhatSystemdQuoteWrites(t *testing.T) {
	for _, path := range []string{"/opt/pwikit/pwikit", "/www/wwwroot/my wiki/pwikit", `/srv/50%$off/"q"\x/pwikit`, "/srv/维基/pwikit"} {
		first, rest := systemdFirst(systemdQuote(path) + ` "update" "-auto"`)
		if first != path {
			t.Errorf("systemdFirst(systemdQuote(%q)) = %q, want %q", path, first, path)
		}
		if rest != `"update" "-auto"` {
			t.Errorf("systemdFirst(systemdQuote(%q)) rest = %q, want the arguments", path, rest)
		}
	}
}

func TestPointSystemdAt(t *testing.T) {
	s := Spec{Name: "pwikit", Executable: "/www/wwwroot/pwikit/pwikit", Root: "/www/wwwroot/pwikit",
		UpdateArgs: []string{"update", "-auto", "-service", "pwikit", "-data-dir", "/www/wwwroot/pwikit", "--"}}
	unit, _ := s.SystemdUpdate()
	next, changed := pointSystemdAt(unit, "/var/lib/pwikit/abc/bin/pwikit")
	if !changed {
		t.Fatal("pointSystemdAt() changed = false, want true")
	}
	s.UpdateExecutable = "/var/lib/pwikit/abc/bin/pwikit"
	want, _ := s.SystemdUpdate()
	if next != want {
		t.Errorf("pointSystemdAt() = %q, want %q", next, want)
	}
	if _, again := pointSystemdAt(next, "/var/lib/pwikit/abc/bin/pwikit"); again {
		t.Error("pointSystemdAt(already pointed) changed = true, want false")
	}
	if got := systemdProgram(next); got != "/var/lib/pwikit/abc/bin/pwikit" {
		t.Errorf("systemdProgram() = %q, want the new program", got)
	}
}

func TestPointPlistAt(t *testing.T) {
	s := Spec{Name: "pwikit", Executable: "/Users/a/pwikit & co/pwikit", Root: "/Users/a/pwikit",
		UpdateArgs: []string{"update", "-auto"}}
	plist := s.LaunchdUpdate()
	if got := plistFirst(plist); got != s.Executable {
		t.Errorf("plistFirst() = %q, want %q", got, s.Executable)
	}
	next, changed := pointPlistAt(plist, "/Library/Application Support/pwikit/x/bin/pwikit")
	if !changed {
		t.Fatal("pointPlistAt() changed = false, want true")
	}
	s.UpdateExecutable = "/Library/Application Support/pwikit/x/bin/pwikit"
	if want := s.LaunchdUpdate(); next != want {
		t.Errorf("pointPlistAt() = %q, want %q", next, want)
	}
	if !strings.Contains(next, "<string>update</string>") {
		t.Errorf("pointPlistAt() = %q, want the arguments kept", next)
	}
}

func TestSystemdUser(t *testing.T) {
	s := Spec{Name: "pwikit", Executable: "/opt/pwikit/pwikit", Root: "/opt/pwikit", User: "www"}
	if got, _ := systemdValue(s.Systemd(), "User"); got != "www" {
		t.Errorf("systemdValue(User) = %q, want %q", got, "www")
	}
	s.User = ""
	if _, ok := systemdValue(s.Systemd(), "User"); ok {
		t.Error("systemdValue(User) ok = true for a unit without one, want false")
	}
}
