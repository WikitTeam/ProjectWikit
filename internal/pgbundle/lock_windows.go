//go:build windows

package pgbundle

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

var errLocked = errors.New("locked")

// Windows locks are mandatory, so the locked byte lies past the end of the file.
const lockOffset = 1 << 40

func lockFD(f *os.File) error {
	ol := &windows.Overlapped{Offset: uint32(lockOffset & 0xffffffff), OffsetHigh: uint32(lockOffset >> 32)}
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return errLocked
	}
	return err
}
