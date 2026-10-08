//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/update"
)

const testAccount = 4242

func asRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("needs root")
	}
}

func ownerUID(t *testing.T, path string) uint32 {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%s) err = %v, want nil", path, err)
	}
	return info.Sys().(*syscall.Stat_t).Uid
}

func instanceLike(t *testing.T) (*paths.Paths, string) {
	t.Helper()
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "pwikit")
	for _, dir := range []string{root, filepath.Join(root, "pgdata"), filepath.Join(root, "files"), filepath.Join(root, "secrets")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	exe := filepath.Join(root, "pwikit")
	for _, file := range []string{exe, filepath.Join(root, "pwikit.toml"), filepath.Join(root, "LICENSE"), filepath.Join(root, "pgdata", "PG_VERSION")} {
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{root, exe, filepath.Join(root, "pwikit.toml"), filepath.Join(root, "pgdata"), filepath.Join(root, "pgdata", "PG_VERSION")} {
		if err := os.Lchown(path, testAccount, testAccount); err != nil {
			t.Fatal(err)
		}
	}
	p, err := paths.New(root)
	if err != nil {
		t.Fatal(err)
	}
	return p, exe
}

func TestProtectInstanceHandsTheTopToRoot(t *testing.T) {
	asRoot(t)
	p, exe := instanceLike(t)
	if exposed := protectInstance(p, exe); exposed != "" {
		t.Fatalf("protectInstance() = %q, want empty", exposed)
	}
	for _, path := range []string{p.Root(), exe, p.Config()} {
		if got := ownerUID(t, path); got != 0 {
			t.Errorf("owner of %s = %d, want 0", path, got)
		}
	}
	if got := ownerUID(t, filepath.Join(p.PGData(), "PG_VERSION")); got != testAccount {
		t.Errorf("owner of pgdata/PG_VERSION = %d, want %d", got, testAccount)
	}
	for _, name := range accountDirs {
		if got := ownerUID(t, filepath.Join(p.Root(), name)); name != "files" && got != testAccount {
			t.Errorf("owner of %s = %d, want %d", name, got, testAccount)
		}
	}
	if uid, _, ok := update.Account(p.Root()); !ok || uid != testAccount {
		t.Errorf("Account() = %d, %t, want %d, true", uid, ok, testAccount)
	}
	if exposed := protectInstance(p, exe); exposed != "" {
		t.Errorf("protectInstance() second time = %q, want empty", exposed)
	}
}

func TestProtectInstanceLeavesADirectoryWithOtherFiles(t *testing.T) {
	asRoot(t)
	p, exe := instanceLike(t)
	if err := os.WriteFile(filepath.Join(p.Root(), "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	exposed := protectInstance(p, exe)
	if !strings.Contains(exposed, "index.php") {
		t.Errorf("protectInstance() = %q, want it to name index.php", exposed)
	}
	for _, path := range []string{p.Root(), exe, p.Config()} {
		if got := ownerUID(t, path); got != testAccount {
			t.Errorf("owner of %s = %d, want %d", path, got, testAccount)
		}
	}
}

func TestExposureOfAProgramOutside(t *testing.T) {
	asRoot(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "pwikit")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := exposure(exe); got != "" {
		t.Errorf("exposure(root's) = %q, want empty", got)
	}
	if err := os.Lchown(exe, testAccount, testAccount); err != nil {
		t.Fatal(err)
	}
	if got := exposure(exe); got == "" {
		t.Error("exposure(another account's) = empty, want a reason")
	}
}

func TestReleaseInstanceGivesTheTopBack(t *testing.T) {
	asRoot(t)
	p, exe := instanceLike(t)
	protectInstance(p, exe)
	releaseInstance(p.Root(), exe, testAccount, testAccount)
	for _, path := range []string{p.Root(), exe, p.Config()} {
		if got := ownerUID(t, path); got != testAccount {
			t.Errorf("owner of %s = %d, want %d", path, got, testAccount)
		}
	}
}

func TestSettingsFileOpenTo(t *testing.T) {
	asRoot(t)
	p, exe := instanceLike(t)
	if got := settingsFileOpenTo(p.Config()); got == "" {
		t.Error(`settingsFileOpenTo(account's) = "", want a reason`)
	}
	protectInstance(p, exe)
	if got := settingsFileOpenTo(p.Config()); got != "" {
		t.Errorf(`settingsFileOpenTo(after protectInstance) = %q, want ""`, got)
	}
}
