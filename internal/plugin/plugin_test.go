package plugin

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func buildHello(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "hello")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := filepath.Abs(filepath.Join("testdata", "hello"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", filepath.Join(dir, "plugin.wasm"), ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("build the test plugin: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(filepath.Join(src, "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(dir)
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestLoadRunsModule(t *testing.T) {
	root := buildHello(t)
	ctx := context.Background()
	h, err := Load(ctx, root, nil, quiet(), func(string) bool { return false })
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	defer h.Close(ctx)
	p := h.modules["echo"]
	if p == nil {
		t.Fatalf("modules[echo] = nil, want hello")
	}
	if !h.ModuleHasContent("hello") {
		t.Errorf("ModuleHasContent(hello) = false, want true")
	}
	if h.ModuleHasContent("echo") {
		t.Errorf("ModuleHasContent(echo) = true, want false")
	}
	site := &db.Site{ID: 1, Slug: "demo"}
	for range 3 {
		out, err := h.call(ctx, p, &callInfo{plugin: p.Name, site: site}, "module",
			[]byte(`{"module":"echo","params":{"text":"hi"},"body":"there"}`))
		if err != nil {
			t.Fatalf("call(echo) err = %v", err)
		}
		if string(out) != "hi|there" {
			t.Errorf("call(echo) = %q, want %q", out, "hi|there")
		}
	}
}

func TestCallStopsAtTimeout(t *testing.T) {
	root := buildHello(t)
	ctx := context.Background()
	h, err := Load(ctx, root, nil, quiet(), func(string) bool { return false })
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	defer h.Close(ctx)
	p := h.modules["spin"]
	start := time.Now()
	_, err = h.call(ctx, p, &callInfo{plugin: p.Name, site: &db.Site{ID: 1}}, "module", []byte(`{"module":"spin"}`))
	if err == nil {
		t.Fatalf("call(spin) err = nil, want a timeout")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("call(spin) took %v, want under 3s", took)
	}
}

func TestLoadSkipsBuiltinClash(t *testing.T) {
	root := buildHello(t)
	ctx := context.Background()
	h, err := Load(ctx, root, nil, quiet(), func(name string) bool { return name == "echo" })
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	defer h.Close(ctx)
	if got := len(h.Installed()); got != 0 {
		t.Errorf("len(Installed()) = %d, want 0", got)
	}
}

func TestReadManifestRefuses(t *testing.T) {
	cases := map[string]string{
		"name":       `{"name":"Bad Name","api":1}`,
		"api":        `{"name":"ok","api":99}`,
		"permission": `{"name":"ok","api":1,"permissions":["sql"]}`,
		"event":      `{"name":"ok","api":1,"events":["user.deleted"]}`,
		"hosts":      `{"name":"ok","api":1,"allowed_hosts":["example.com"]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := readManifest(dir); err == nil || strings.TrimSpace(err.Error()) == "" {
				t.Errorf("readManifest(%s) err = %v, want an error", raw, err)
			}
		})
	}
}

func TestReadSettings(t *testing.T) {
	dir := t.TempDir()
	got, err := readSettings(dir)
	if err != nil || got.Config != nil {
		t.Fatalf("readSettings(empty) = %+v, %v, want zero and nil", got, err)
	}
	raw := `{"config":{"url":"http://127.0.0.1:8100"},"allowed_hosts":["127.0.0.1"]}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = readSettings(dir)
	if err != nil {
		t.Fatalf("readSettings() err = %v", err)
	}
	if got.Config["url"] != "http://127.0.0.1:8100" {
		t.Errorf("Config[url] = %q, want http://127.0.0.1:8100", got.Config["url"])
	}
	if len(got.AllowedHosts) != 1 || got.AllowedHosts[0] != "127.0.0.1" {
		t.Errorf("AllowedHosts = %v, want [127.0.0.1]", got.AllowedHosts)
	}
}
