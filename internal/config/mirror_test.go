package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMirror = "https://mirror.example.org/projwikit/update"

func TestSetUpdateMirrorWritesTheTemplateFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pwikit.toml")
	if err := SetUpdateMirror(path, testMirror); err != nil {
		t.Fatalf("SetUpdateMirror() err = %v, want nil", err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatalf("Load() err = %v, want nil", err)
	}
	if f.Update.Mirror != testMirror {
		t.Errorf("Load().Update.Mirror = %q, want %q", f.Update.Mirror, testMirror)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "# min_age = \"24h\"") {
		t.Errorf("SetUpdateMirror() file = %q, want the template comments kept", raw)
	}
	if strings.Contains(string(raw), `# mirror = ""`) {
		t.Errorf("SetUpdateMirror() file still holds the commented example")
	}
}

func TestSetMirrorLine(t *testing.T) {
	cases := []struct {
		name, in, mirror, want string
	}{
		{
			"replaces a set value",
			"[update]\nmirror = \"https://old.example.org\"\n",
			testMirror,
			"[update]\nmirror = \"" + testMirror + "\"\n",
		},
		{
			"takes the place of the example",
			"[update]\n# check = true\n# mirror = \"\"\n",
			testMirror,
			"[update]\n# check = true\nmirror = \"" + testMirror + "\"\n",
		},
		{
			"goes under an update section without one",
			"[update]\nauto = false\n",
			testMirror,
			"[update]\nmirror = \"" + testMirror + "\"\nauto = false\n",
		},
		{
			"adds the section when there is none",
			"[server]\nlisten = \":80\"\n\n",
			testMirror,
			"[server]\nlisten = \":80\"\n\n[update]\nmirror = \"" + testMirror + "\"\n",
		},
		{
			"leaves a mirror key in another section alone",
			"[other]\n# mirror = \"\"\n[update]\n# mirror = \"\"\n",
			testMirror,
			"[other]\n# mirror = \"\"\n[update]\nmirror = \"" + testMirror + "\"\n",
		},
		{
			"clears back to the example",
			"[update]\nmirror = \"" + testMirror + "\"\n",
			"",
			"[update]\n# mirror = \"\"\n",
		},
		{
			"drops a trailing slash",
			"[update]\n",
			testMirror + "/",
			"[update]\nmirror = \"" + testMirror + "\"\n",
		},
		{
			"keeps windows line endings",
			"[update]\r\n# mirror = \"\"\r\n",
			testMirror,
			"[update]\r\nmirror = \"" + testMirror + "\"\r\n",
		},
	}
	for _, c := range cases {
		if got := setMirrorLine(c.in, c.mirror); got != c.want {
			t.Errorf("setMirrorLine(%s) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCheckMirror(t *testing.T) {
	cases := []struct {
		mirror string
		ok     bool
	}{
		{"", true},
		{testMirror, true},
		{"http://10.0.0.2:8080/update", true},
		{"ftp://mirror.example.org", false},
		{"mirror.example.org/update", false},
		{"https://mirror.example.org/a b", false},
		{`https://mirror.example.org/"`, false},
	}
	for _, c := range cases {
		if err := CheckMirror(c.mirror); (err == nil) != c.ok {
			t.Errorf("CheckMirror(%q) err = %v, want ok %v", c.mirror, err, c.ok)
		}
	}
}
