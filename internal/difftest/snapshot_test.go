package difftest

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	EnvSnapshot       = "PWIKIT_SNAPSHOT"
	EnvSnapshotUpdate = "PWIKIT_SNAPSHOT_UPDATE"
)

func TestSnapshotRoundTrip(t *testing.T) {
	want := Response{
		Status: 200,
		Header: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
			"Set-Cookie":   []string{"a=1", "b=2"},
		},
		Body: []byte("<p>one</p>\n\n<p>two</p>\n"),
	}
	got, err := DecodeSnapshot(EncodeSnapshot(want))
	if err != nil {
		t.Fatalf("DecodeSnapshot(EncodeSnapshot(want)) err = %v, want nil", err)
	}
	if got.Status != want.Status {
		t.Errorf("Status = %d, want %d", got.Status, want.Status)
	}
	if string(got.Body) != string(want.Body) {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
	if got.Header.Get("Content-Type") != want.Header.Get("Content-Type") {
		t.Errorf("Content-Type = %q, want %q", got.Header.Get("Content-Type"), want.Header.Get("Content-Type"))
	}
	if len(got.Header.Values("Set-Cookie")) != 2 {
		t.Errorf("len(Set-Cookie) = %d, want 2", len(got.Header.Values("Set-Cookie")))
	}
}

func TestDecodeSnapshotRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"no blank line":          "200\nContent-Type: text/html\n",
		"no status":              "hello\n\nbody",
		"header without a colon": "200\nContent-Type\n\nbody",
	}
	for name, raw := range cases {
		if _, err := DecodeSnapshot([]byte(raw)); err == nil {
			t.Errorf("DecodeSnapshot(%s) err = nil, want an error", name)
		}
	}
}

func TestSnapshotPathSeparatesTargetsDifferingOnlyInCase(t *testing.T) {
	lower := SnapshotPath("default", Request{Method: http.MethodGet, Target: "/-/users/probe"})
	upper := SnapshotPath("default", Request{Method: http.MethodGet, Target: "/-/users/PROBE"})
	if strings.EqualFold(lower, upper) {
		t.Errorf("SnapshotPath of /-/users/probe = %q, want it to differ from %q by more than case", lower, upper)
	}
	if dir := filepath.Dir(lower); dir != filepath.Join("testdata", "snapshots", "default") {
		t.Errorf("filepath.Dir(SnapshotPath) = %q, want %q", dir, filepath.Join("testdata", "snapshots", "default"))
	}
	if filepath.Ext(lower) != ".golden" {
		t.Errorf("filepath.Ext(SnapshotPath) = %q, want %q", filepath.Ext(lower), ".golden")
	}
}

func TestCorpusSnapshot(t *testing.T) {
	base := os.Getenv(EnvSnapshot)
	if base == "" {
		t.Skipf("%s not set, skipping the snapshot run", EnvSnapshot)
	}
	runner, err := NewSnapshotRunner(base)
	if err != nil {
		t.Fatalf("NewSnapshotRunner(%q) err = %v, want nil", base, err)
	}
	runner.Host = os.Getenv(EnvHost)

	corpus, err := ParseCorpus(loadCorpus(t))
	if err != nil {
		t.Fatalf("ParseCorpus() err = %v, want nil", err)
	}
	set := snapshotSet()
	update := os.Getenv(EnvSnapshotUpdate) != ""

	for _, req := range corpus {
		t.Run(req.String(), func(t *testing.T) {
			got, err := runner.fetch(context.Background(), runner.A, req)
			if err != nil {
				t.Fatalf("fetch(%s) err = %v, want nil", req, err)
			}
			path := SnapshotPath(set, req)
			if update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("MkdirAll(%q) err = %v, want nil", filepath.Dir(path), err)
				}
				if err := os.WriteFile(path, EncodeSnapshot(got), 0o644); err != nil {
					t.Fatalf("WriteFile(%q) err = %v, want nil", path, err)
				}
				return
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%q) err = %v, want nil; record it with %s=1", path, err, EnvSnapshotUpdate)
			}
			want, err := DecodeSnapshot(raw)
			if err != nil {
				t.Fatalf("DecodeSnapshot(%q) err = %v, want nil", path, err)
			}
			if result := runner.Comparer.Compare(got, want); !result.Same() {
				t.Errorf("%s: a = got, b = want:\n%s", req, result)
			}
		})
	}
}

func snapshotSet() string {
	path := os.Getenv(EnvCorpus)
	if path == "" {
		return "default"
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
