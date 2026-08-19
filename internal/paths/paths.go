package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvDataDir = "PWIKIT_DATA_DIR"

type Source string

const (
	SourceExplicit   Source = "explicit"
	SourceEnv        Source = "env"
	SourceExecutable Source = "executable"
	SourceGoRun      Source = "go-run"
)

type Paths struct {
	root   string
	source Source
}

func New(override string) (*Paths, error) {
	root, source, err := resolveRoot(override)
	if err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("状态目录 %q 无法转为绝对路径: %w", root, err)
	}

	return &Paths{root: abs, source: source}, nil
}

func resolveRoot(override string) (string, Source, error) {
	if override != "" {
		return override, SourceExplicit, nil
	}
	if env := os.Getenv(EnvDataDir); env != "" {
		return env, SourceEnv, nil
	}
	return executableDir()
}

func executableDir() (string, Source, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("定位可执行文件失败: %w", err)
	}

	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	dir := filepath.Dir(exe)
	if isGoRunTemp(dir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("go run 下取当前目录失败: %w", err)
		}
		return cwd, SourceGoRun, nil
	}
	return dir, SourceExecutable, nil
}

func isGoRunTemp(dir string) bool {
	if !strings.Contains(dir, "go-build") {
		return false
	}

	tmp := os.TempDir()

	if resolved, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = resolved
	}

	rel, err := filepath.Rel(tmp, dir)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (p *Paths) Root() string   { return p.root }
func (p *Paths) Source() Source { return p.source }

func (p *Paths) Config() string   { return filepath.Join(p.root, "pwikit.toml") }
func (p *Paths) Files() string    { return filepath.Join(p.root, "files") }
func (p *Paths) Archive() string  { return filepath.Join(p.root, "archive") }
func (p *Paths) Secrets() string  { return filepath.Join(p.root, "secrets") }
func (p *Paths) PGData() string   { return filepath.Join(p.root, "pgdata") }
func (p *Paths) Postgres() string { return filepath.Join(p.root, "postgres") }

func (p *Paths) Certs() string { return filepath.Join(p.Secrets(), "certs") }

func (p *Paths) EnsureBase() error {
	if err := os.MkdirAll(p.Secrets(), 0o700); err != nil {
		return fmt.Errorf("建立 %s 失败: %w", p.Secrets(), err)
	}
	for _, dir := range []string{p.Files(), p.Archive()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("建立 %s 失败: %w", dir, err)
		}
	}
	return nil
}

var ErrEscapes = errors.New("路径越界")

func Resolve(base, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %q 是绝对路径", ErrEscapes, rel)
	}

	joined := filepath.Join(base, rel)

	relToBase, err := filepath.Rel(base, joined)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrEscapes, rel)
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q", ErrEscapes, rel)
	}
	return joined, nil
}
