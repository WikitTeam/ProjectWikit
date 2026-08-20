package difftest

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	EnvA      = "PWIKIT_DIFF_A"
	EnvB      = "PWIKIT_DIFF_B"
	EnvHost   = "PWIKIT_DIFF_HOST"
	EnvCorpus = "PWIKIT_DIFF_CORPUS"
	EnvDump   = "PWIKIT_DIFF_DUMP"
)

//go:embed corpus/default.txt
var defaultCorpus string

func TestDefaultCorpusParses(t *testing.T) {
	corpus, err := ParseCorpus(defaultCorpus)
	if err != nil {
		t.Fatalf("ParseCorpus(defaultCorpus) err = %v, want nil", err)
	}
	if len(corpus) == 0 {
		t.Fatal("len(corpus) = 0, want more than 0")
	}
}

func TestCorpusDiff(t *testing.T) {
	a, b := os.Getenv(EnvA), os.Getenv(EnvB)
	if a == "" || b == "" {
		t.Skipf("%s / %s not set, skipping the differential run", EnvA, EnvB)
	}

	runner, err := NewRunner(a, b)
	if err != nil {
		t.Fatalf("NewRunner(%q, %q) err = %v, want nil", a, b, err)
	}
	runner.Host = os.Getenv(EnvHost)
	if runner.Host == "" {
		t.Logf("%s is unset: the two sides receive different Host headers, "+
			"which changes Host to Site resolution on the Django side", EnvHost)
	}

	corpus, err := ParseCorpus(loadCorpus(t))
	if err != nil {
		t.Fatalf("ParseCorpus() err = %v, want nil", err)
	}

	dump := os.Getenv(EnvDump)
	scrubs := make(map[string]int)
	known := 0

	for _, req := range corpus {
		t.Run(req.String(), func(t *testing.T) {
			result, respA, respB, err := runner.Do(context.Background(), req)
			if err != nil {
				t.Fatalf("Do(%s) err = %v, want nil", req, err)
			}
			for name, n := range result.Scrubs {
				scrubs[name] += n
			}
			if result.Same() {
				return
			}
			if dump != "" {
				writeDump(t, dump, req, respA, respB)
			}
			if req.KnownDiffers {
				known++
				t.Logf("known difference:\n%s", result)
				return
			}
			t.Errorf("%s differs:\n%s", req, result)
		})
	}

	for name, n := range scrubs {
		t.Logf("scrubber %q fired %d times", name, n)
	}
	if known > 0 {
		t.Logf("%d known differences were reported but not failed; "+
			"each one needs a reason in the TODO ledger of PROGRESS.md", known)
	}
}

func loadCorpus(t *testing.T) string {
	t.Helper()
	path := os.Getenv(EnvCorpus)
	if path == "" {
		return defaultCorpus
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) err = %v, want nil", path, err)
	}
	return string(raw)
}

func writeDump(t *testing.T, dir string, req Request, a, b Response) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) err = %v, want nil", dir, err)
	}
	stem := filepath.Join(dir, dumpName(req))
	for suffix, body := range map[string][]byte{".a.html": a.Body, ".b.html": b.Body} {
		if err := os.WriteFile(stem+suffix, body, 0o644); err != nil {
			t.Fatalf("WriteFile(%q) err = %v, want nil", stem+suffix, err)
		}
	}
	t.Logf("bodies written to %s.a.html and %s.b.html", stem, stem)
}

func dumpName(req Request) string {
	name := req.Method + strings.ReplaceAll(req.Target, "/", "_")
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-' || r == '.':
			return r
		}
		return '-'
	}, name)
}
