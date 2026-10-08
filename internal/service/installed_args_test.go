package service

import (
	"strings"
	"testing"
)

func TestDataDirArg(t *testing.T) {
	cases := map[string][]string{
		"/www/wwwroot/pwikit": {"/var/lib/pwikit/ab/bin/pwikit", "serve", "-data-dir", "/www/wwwroot/pwikit"},
		"/opt/pwikit":         {"pwikit", "serve", "-data-dir=/opt/pwikit"},
		"/srv/p":              {"pwikit", "serve", "--data-dir", "/srv/p"},
		"":                    {"pwikit", "serve"},
	}
	for want, args := range cases {
		if got := dataDirArg(args); got != want {
			t.Errorf("dataDirArg(%q) = %q, want %q", args, got, want)
		}
	}
}

func TestSystemdArgs(t *testing.T) {
	unit := "[Service]\nExecStart=\"/var/lib/pwikit/ab/bin/pwikit\" serve -data-dir \"/www/www root/pwikit\"\n"
	if got := dataDirArg(systemdArgs(unit)); got != "/www/www root/pwikit" {
		t.Errorf("dataDirArg(systemdArgs()) = %q, want %q", got, "/www/www root/pwikit")
	}
}

func TestPlistArgs(t *testing.T) {
	text := "<key>ProgramArguments</key>\n\t<array>\n\t\t<string>/x/pwikit</string>\n\t\t<string>serve</string>\n\t\t<string>-data-dir</string>\n\t\t<string>/a &amp; b</string>\n\t</array>"
	if got := dataDirArg(plistArgs(text)); got != "/a & b" {
		t.Errorf("dataDirArg(plistArgs()) = %q, want %q", got, "/a & b")
	}
}

func TestCommandLineRunsTheRootCopy(t *testing.T) {
	s := Spec{Executable: "/www/pwikit/pwikit", RunExecutable: "/var/lib/pwikit/ab/bin/pwikit", Args: []string{"serve"}}
	if got := s.CommandLine()[0]; got != s.RunExecutable {
		t.Errorf("CommandLine()[0] = %q, want %q", got, s.RunExecutable)
	}
	s.RunExecutable = ""
	if got := s.CommandLine()[0]; got != s.Executable {
		t.Errorf("CommandLine()[0] = %q, want %q", got, s.Executable)
	}
}

func TestServeExtra(t *testing.T) {
	args := []string{"/opt/pwikit/pwikit", "serve", "-data-dir", "/opt/pwikit", "-log-file", "/x.log", "-listen", ":80", "-domain=wiki.example"}
	got := serveExtra(args)
	want := []string{"-listen", ":80", "-domain=wiki.example"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("serveExtra(%q) = %q, want %q", args, got, want)
	}
}
