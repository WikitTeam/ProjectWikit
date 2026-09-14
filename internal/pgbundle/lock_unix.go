//go:build unix

package pgbundle

import (
	"errors"
	"os"
	"syscall"
)

var errLocked = errors.New("locked")

func lockFD(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return errLocked
	}
	return err
}
