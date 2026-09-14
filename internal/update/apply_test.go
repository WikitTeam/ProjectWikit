package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

type fakeInstance struct {
	t       *testing.T
	root    string
	exe     string
	pgdata  string
	stopped bool
	starts  int

	healthy   func(version string) bool
	onStart   func(version string)
	preflight Preflight
	runErr    map[string]error
	ran       []string
	health    *httptest.Server
}

func newFakeInstance(t *testing.T) *fakeInstance {
	t.Helper()
	root := t.TempDir()
	f := &fakeInstance{
		t:       t,
		root:    root,
		exe:     filepath.Join(root, ExecutableName(runtime.GOOS)),
		pgdata:  filepath.Join(root, "pgdata"),
		healthy: func(string) bool { return true },
		runErr:  map[string]error{},
	}
	must(t, os.WriteFile(f.exe, []byte("old program"), 0o755))
	must(t, os.MkdirAll(f.pgdata, 0o755))
	must(t, os.WriteFile(filepath.Join(f.pgdata, "table"), []byte("before"), 0o644))
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fakeInstance) StopService(context.Context) error {
	f.stopped = true
	return nil
}

func (f *fakeInstance) StartService(context.Context) error {
	f.stopped = false
	f.starts++
	body, err := os.ReadFile(f.exe)
	if err != nil {
		return err
	}
	running := "v1.0.0"
	if string(body) == "new program" {
		running = "v1.1.0"
	}
	if f.onStart != nil {
		f.onStart(running)
	}
	if f.health != nil {
		f.health.Close()
	}
	ok := f.healthy(running)
	f.health = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := Health{Version: running, Database: true, Page: http.StatusOK}
		if !ok {
			h.Page = http.StatusInternalServerError
		}
		json.NewEncoder(w).Encode(h)
	}))
	f.t.Cleanup(f.health.Close)
	addr := f.health.Listener.Addr().(*net.TCPAddr).String()
	return WriteRuntime(filepath.Join(f.root, "update"), Runtime{Version: running, StartedAt: time.Now().Add(time.Second), Health: addr})
}

func (f *fakeInstance) run(_ context.Context, exe string, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")
	f.ran = append(f.ran, joined)
	for prefix, err := range f.runErr {
		if strings.HasPrefix(joined, prefix) {
			return nil, err
		}
	}
	if strings.HasPrefix(joined, "update preflight") {
		body, _ := json.Marshal(f.preflight)
		return body, nil
	}
	return nil, nil
}

func releaseServer(t *testing.T) (*httptest.Server, Target) {
	t.Helper()
	name := ExecutableName(runtime.GOOS)
	dir := PackageDir("v1.1.0", runtime.GOOS, runtime.GOARCH)
	file := PackageFile("v1.1.0", runtime.GOOS, runtime.GOARCH)
	var pkg bytes.Buffer
	if runtime.GOOS == "windows" {
		zw := zip.NewWriter(&pkg)
		w, err := zw.Create(dir + "/" + name)
		must(t, err)
		w.Write([]byte("new program"))
		must(t, zw.Close())
	} else {
		gz := gzip.NewWriter(&pkg)
		tw := tar.NewWriter(gz)
		must(t, tw.WriteHeader(&tar.Header{Name: dir + "/" + name, Mode: 0o755, Size: int64(len("new program")), Typeflag: tar.TypeReg}))
		tw.Write([]byte("new program"))
		must(t, tw.Close())
		must(t, gz.Close())
	}
	sum := sha256.Sum256(pkg.Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download/v1.1.0/"+file {
			w.Write(pkg.Bytes())
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server, Target{Version: "v1.1.0", File: file, SHA256: hex.EncodeToString(sum[:])}
}

func (f *fakeInstance) applier(releases string) *Applier {
	return &Applier{
		Root: f.root, PGData: f.pgdata, Executable: f.exe, Bundled: true, Current: "v1.0.0",
		Source:       Source{Releases: releases},
		Machine:      f,
		Run:          f.run,
		HealthWithin: 5 * time.Second,
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", path, err)
	}
	return string(body)
}

func TestApplyInstallsARelease(t *testing.T) {
	f := newFakeInstance(t)
	server, target := releaseServer(t)
	f.preflight = Preflight{Version: "v1.1.0"}

	out := f.applier(server.URL).Apply(context.Background(), target)
	if out.Err != nil {
		t.Fatalf("Apply() err = %v, want nil", out.Err)
	}
	if got := readFile(t, f.exe); got != "new program" {
		t.Errorf("executable after Apply() = %q, want the new program", got)
	}
	if out.Kind != RollbackNone {
		t.Errorf("Apply().Kind = %q, want %q", out.Kind, RollbackNone)
	}
	if got := readFile(t, filepath.Join(RollbackDir(f.root), ExecutableName(runtime.GOOS))); got != "old program" {
		t.Errorf("rollback point program = %q, want the old program", got)
	}
	if f.stopped {
		t.Error("service after Apply() stopped = true, want it running")
	}
}

func TestApplyRollsBackAnUnhealthyRelease(t *testing.T) {
	f := newFakeInstance(t)
	server, target := releaseServer(t)
	f.preflight = Preflight{Version: "v1.1.0"}
	f.healthy = func(version string) bool { return version != "v1.1.0" }

	out := f.applier(server.URL).Apply(context.Background(), target)
	if out.Err == nil || !out.RolledBack {
		t.Fatalf("Apply() = %+v, want an error and a rollback", out)
	}
	if out.RollbackErr != nil {
		t.Errorf("Apply().RollbackErr = %v, want nil", out.RollbackErr)
	}
	if got := readFile(t, f.exe); got != "old program" {
		t.Errorf("executable after the rollback = %q, want the old program", got)
	}
	if f.starts != 2 {
		t.Errorf("service starts = %d, want 2", f.starts)
	}
}

func TestApplyPutsBackTheDataABreakingReleaseChanged(t *testing.T) {
	f := newFakeInstance(t)
	server, target := releaseServer(t)
	f.preflight = Preflight{Version: "v1.1.0", Pending: []string{"0011_x.sql"}, PendingBreaking: true}
	f.healthy = func(version string) bool { return version != "v1.1.0" }
	f.onStart = func(version string) {
		if version == "v1.1.0" {
			os.WriteFile(filepath.Join(f.pgdata, "table"), []byte("migrated"), 0o644)
		}
	}

	out := f.applier(server.URL).Apply(context.Background(), target)
	if out.Kind != RollbackPGData || !out.RolledBack || out.RollbackErr != nil {
		t.Fatalf("Apply() = %+v, want a pgdata rollback that succeeded", out)
	}
	if got := readFile(t, filepath.Join(f.pgdata, "table")); got != "before" {
		t.Errorf("pgdata after the rollback = %q, want %q", got, "before")
	}
}

func TestApplyTouchesNothingWhenThePreflightObjects(t *testing.T) {
	f := newFakeInstance(t)
	server, target := releaseServer(t)
	f.preflight = Preflight{Version: "v1.1.0", Problem: "the database was upgraded by a newer pwikit"}

	out := f.applier(server.URL).Apply(context.Background(), target)
	if out.Err == nil || out.RolledBack {
		t.Fatalf("Apply() = %+v, want an error without a rollback", out)
	}
	if f.stopped || f.starts != 0 {
		t.Errorf("service stopped=%t starts=%d, want it never stopped", f.stopped, f.starts)
	}
	if got := readFile(t, f.exe); got != "old program" {
		t.Errorf("executable = %q, want the old program", got)
	}
}

func TestApplyRestartsTheOldReleaseWhenTheRollbackPointFails(t *testing.T) {
	f := newFakeInstance(t)
	server, target := releaseServer(t)
	f.preflight = Preflight{Version: "v1.1.0", PendingBreaking: true}
	f.runErr["backup create"] = errors.New("disk full")
	a := f.applier(server.URL)
	a.Bundled = false

	out := a.Apply(context.Background(), target)
	if out.Err == nil || out.RolledBack || out.RollbackErr != nil {
		t.Fatalf("Apply() = %+v, want an error with the old release running again", out)
	}
	if got := readFile(t, f.exe); got != "old program" {
		t.Errorf("executable = %q, want the old program", got)
	}
	if f.stopped || f.starts != 1 {
		t.Errorf("service stopped=%t starts=%d, want it started once again", f.stopped, f.starts)
	}
	if !slices.ContainsFunc(f.ran, func(s string) bool { return strings.HasPrefix(s, "backup create") }) {
		t.Errorf("commands run = %v, want a backup attempted", f.ran)
	}
}

func TestManualRollbackPutsBackTheRelease(t *testing.T) {
	f := newFakeInstance(t)
	server, target := releaseServer(t)
	f.preflight = Preflight{Version: "v1.1.0"}
	a := f.applier(server.URL)
	if out := a.Apply(context.Background(), target); out.Err != nil {
		t.Fatalf("Apply() err = %v, want nil", out.Err)
	}

	a.Current = "v1.1.0"
	point, err := a.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback() err = %v, want nil", err)
	}
	if point.From != "v1.0.0" {
		t.Errorf("Rollback().From = %q, want v1.0.0", point.From)
	}
	if got := readFile(t, f.exe); got != "old program" {
		t.Errorf("executable after Rollback() = %q, want the old program", got)
	}
}
