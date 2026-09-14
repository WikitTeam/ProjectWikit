//go:build !windows

package shellpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func program(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, command)
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func linkTarget(t *testing.T, link string) string {
	t.Helper()
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink(%s) err = %v, want nil", link, err)
	}
	return target
}

func TestInstallCreatesTheLink(t *testing.T) {
	root := t.TempDir()
	exe := program(t, filepath.Join(root, "opt"))
	bin := filepath.Join(root, "bin")

	place, outcome, err := Install(exe, bin, false)
	if err != nil {
		t.Fatalf("Install() err = %v, want nil", err)
	}
	if outcome != Created {
		t.Errorf("Install() outcome = %q, want %q", outcome, Created)
	}
	if !place.Ours || !place.Working {
		t.Errorf("Install() place = %+v, want ours and working", place)
	}
	if got := linkTarget(t, filepath.Join(bin, command)); got != exe {
		t.Errorf("link target = %q, want %q", got, exe)
	}

	_, outcome, err = Install(exe, bin, false)
	if err != nil || outcome != Unchanged {
		t.Errorf("Install() again = %q, %v, want %q, nil", outcome, err, Unchanged)
	}
}

func TestInstallReplacesALinkLeftByAMovedInstance(t *testing.T) {
	root := t.TempDir()
	old := program(t, filepath.Join(root, "old"))
	bin := filepath.Join(root, "bin")
	if _, _, err := Install(old, bin, false); err != nil {
		t.Fatal(err)
	}
	moved := program(t, filepath.Join(root, "new"))
	if err := os.RemoveAll(filepath.Join(root, "old")); err != nil {
		t.Fatal(err)
	}

	_, outcome, err := Install(moved, bin, false)
	if err != nil {
		t.Fatalf("Install(after the move) err = %v, want nil", err)
	}
	if outcome != Replaced {
		t.Errorf("Install(after the move) outcome = %q, want %q", outcome, Replaced)
	}
	if got := linkTarget(t, filepath.Join(bin, command)); got != moved {
		t.Errorf("link target = %q, want %q", got, moved)
	}
}

func TestInstallRefusesAnotherLiveInstance(t *testing.T) {
	root := t.TempDir()
	first := program(t, filepath.Join(root, "first"))
	second := program(t, filepath.Join(root, "second"))
	bin := filepath.Join(root, "bin")
	if _, _, err := Install(first, bin, false); err != nil {
		t.Fatal(err)
	}

	_, _, err := Install(second, bin, false)
	if err == nil || !strings.Contains(err.Error(), "-force") {
		t.Errorf("Install(second) err = %v, want one naming -force", err)
	}
	if _, _, err := Install(second, bin, true); err != nil {
		t.Fatalf("Install(second, force) err = %v, want nil", err)
	}
	if got := linkTarget(t, filepath.Join(bin, command)); got != second {
		t.Errorf("link target after force = %q, want %q", got, second)
	}
}

func TestInstallRefusesAPlainFile(t *testing.T) {
	root := t.TempDir()
	exe := program(t, filepath.Join(root, "opt"))
	bin := filepath.Join(root, "bin")
	program(t, bin)

	if _, _, err := Install(exe, bin, false); err == nil {
		t.Error("Install(over a plain file) err = nil, want non-nil")
	}
}

func TestUninstallRemovesOnlyItsOwnLink(t *testing.T) {
	root := t.TempDir()
	first := program(t, filepath.Join(root, "first"))
	second := program(t, filepath.Join(root, "second"))
	bin := filepath.Join(root, "bin")
	if _, _, err := Install(first, bin, false); err != nil {
		t.Fatal(err)
	}

	_, outcome, err := Uninstall(second, bin)
	if err != nil || outcome != Absent {
		t.Errorf("Uninstall(another instance) = %q, %v, want %q, nil", outcome, err, Absent)
	}
	if _, err := os.Lstat(filepath.Join(bin, command)); err != nil {
		t.Errorf("link after another instance uninstalled err = %v, want it kept", err)
	}

	_, outcome, err = Uninstall(first, bin)
	if err != nil || outcome != Removed {
		t.Errorf("Uninstall(own) = %q, %v, want %q, nil", outcome, err, Removed)
	}
	if _, err := os.Lstat(filepath.Join(bin, command)); !os.IsNotExist(err) {
		t.Errorf("link after uninstall err = %v, want it gone", err)
	}
}

func TestStatusReportsALinkThatLeadsNowhere(t *testing.T) {
	root := t.TempDir()
	exe := program(t, filepath.Join(root, "opt"))
	bin := filepath.Join(root, "bin")
	if _, _, err := Install(exe, bin, false); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "opt")); err != nil {
		t.Fatal(err)
	}

	places, err := Status(exe, bin)
	if err != nil {
		t.Fatalf("Status() err = %v, want nil", err)
	}
	if len(places) != 1 || places[0].Working {
		t.Errorf("Status() = %+v, want one place that leads nowhere", places)
	}
}
