//go:build windows

package main

import (
	"context"
	"os/exec"

	"golang.org/x/sys/windows"
)

func runOwner(ctx context.Context, root, exe string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}

func asOwner(string) (bool, error) { return false, nil }

func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259
}
