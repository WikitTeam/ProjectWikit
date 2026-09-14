package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewPrefersExplicitOverEnv(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	explicit := t.TempDir()

	p, err := New(explicit)
	if err != nil {
		t.Fatal(err)
	}
	if p.Root() != explicit {
		t.Errorf("Root() = %q, want %q", p.Root(), explicit)
	}
	if p.Source() != SourceExplicit {
		t.Errorf("Source() = %q, want %q", p.Source(), SourceExplicit)
	}
}

func TestNewFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)

	p, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Root() != dir {
		t.Errorf("Root() = %q, want %q", p.Root(), dir)
	}
	if p.Source() != SourceEnv {
		t.Errorf("Source() = %q, want %q", p.Source(), SourceEnv)
	}
}

func TestNewMakesRootAbsolute(t *testing.T) {
	p, err := New("data")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(p.Root()) {
		t.Errorf("filepath.IsAbs(Root()) = false, Root() = %q", p.Root())
	}
}

func TestLayoutIsFlat(t *testing.T) {
	root := t.TempDir()
	p, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"Config":   p.Config(),
		"Files":    p.Files(),
		"Archive":  p.Archive(),
		"Secrets":  p.Secrets(),
		"PGData":   p.PGData(),
		"Postgres": p.Postgres(),
		"Locales":  p.Locales(),
	}
	for name, got := range cases {
		if filepath.Dir(got) != root {
			t.Errorf("filepath.Dir(%s()) = %q, want %q", name, filepath.Dir(got), root)
		}
	}

	if filepath.Dir(p.Certs()) != p.Secrets() {
		t.Errorf("filepath.Dir(Certs()) = %q, want %q", filepath.Dir(p.Certs()), p.Secrets())
	}
}

func TestEnsureBaseCreatesDirs(t *testing.T) {
	p, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.EnsureBase(); err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{p.Files(), p.Archive(), p.Secrets()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Stat(%s) err = %v, want nil", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s IsDir() = false, want true", dir)
		}
	}

	if _, err := os.Stat(p.PGData()); !os.IsNotExist(err) {
		t.Errorf("EnsureBase() created %s, want it absent", p.PGData())
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(p.Secrets())
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("secrets mode = %o, want 700", perm)
		}
	}
}

func TestEnsureBaseIsIdempotent(t *testing.T) {
	p, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.EnsureBase(); err != nil {
		t.Fatal(err)
	}
	if err := p.EnsureBase(); err != nil {
		t.Errorf("second EnsureBase() err = %v, want nil", err)
	}
}

func TestIsGoRunTemp(t *testing.T) {
	tmp := os.TempDir()
	if resolved, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = resolved
	}

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{
			name: "go run build directory",
			dir:  filepath.Join(tmp, "go-build2451739961", "b001", "exe"),
			want: true,
		},
		{
			name: "normal install location",
			dir:  filepath.Join(tmp, "pwikit"),
			want: false,
		},
		{
			name: "go-build outside the temp directory",
			dir:  filepath.Join("/opt", "go-build", "pwikit"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGoRunTemp(tt.dir); got != tt.want {
				t.Errorf("isGoRunTemp(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestResolveBlocksEscape(t *testing.T) {
	base := filepath.Join(t.TempDir(), "files")

	ok := []struct {
		rel  string
		want string
	}{
		{"a.png", filepath.Join(base, "a.png")},
		{"page/a.png", filepath.Join(base, "page", "a.png")},
		{"page/../a.png", filepath.Join(base, "a.png")},
		{"./a.png", filepath.Join(base, "a.png")},
	}
	for _, tt := range ok {
		got, err := Resolve(base, tt.rel)
		if err != nil {
			t.Errorf("Resolve(%q) err = %v, want nil", tt.rel, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.rel, got, tt.want)
		}
	}

	escapes := []string{
		"../secrets/certs/key.pem",
		"..",
		"page/../../pwikit.toml",
		"a/b/c/../../../../etc/passwd",
	}
	for _, rel := range escapes {
		if got, err := Resolve(base, rel); !errors.Is(err, ErrEscapes) {
			t.Errorf("Resolve(%q) = %q, err = %v, want ErrEscapes", rel, got, err)
		}
	}
}

func TestResolveRejectsAbsolute(t *testing.T) {
	base := t.TempDir()

	abs := "/etc/passwd"
	if runtime.GOOS == "windows" {
		abs = `C:\Windows\System32\config\SAM`
	}
	if _, err := Resolve(base, abs); !errors.Is(err, ErrEscapes) {
		t.Errorf("Resolve(%q) err = %v, want ErrEscapes", abs, err)
	}
}

func TestResolveEscapeReturnValues(t *testing.T) {
	base := filepath.Join(t.TempDir(), "files")
	got, err := Resolve(base, "../../pwikit.toml")
	if err == nil {
		t.Fatalf("Resolve() = %q, err = nil, want ErrEscapes", got)
	}
	if got != "" {
		t.Errorf("Resolve() on error = %q, want empty string", got)
	}
	if !strings.Contains(err.Error(), "pwikit.toml") {
		t.Errorf("err = %v, want substring %q", err, "pwikit.toml")
	}
}
