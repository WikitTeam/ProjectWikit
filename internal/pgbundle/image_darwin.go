package pgbundle

import (
	"bytes"
	"errors"
	"syscall"

	"golang.org/x/sys/unix"
)

func processImage(pid int) (bool, string) {
	if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
		return false, ""
	}
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(raw) < 5 {
		return true, ""
	}
	path := raw[4:]
	if end := bytes.IndexByte(path, 0); end >= 0 {
		path = path[:end]
	}
	return true, string(path)
}
