//go:build unix && !linux && !darwin

package pgbundle

import (
	"errors"
	"syscall"
)

func processImage(pid int) (bool, string) {
	return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH), ""
}
