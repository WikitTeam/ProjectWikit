//go:build windows

package update

import (
	"errors"

	"golang.org/x/sys/windows"
)

func FreeBytes(dir string) (uint64, error) {
	path, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var free, total, all uint64
	if err := windows.GetDiskFreeSpaceEx(path, &free, &total, &all); err != nil {
		return 0, err
	}
	return free, nil
}

var ErrCannotHandOver = errors.New("the filesystem does not let root change who owns a file")

func ChownTree(string, string) error { return nil }

func Owner(string) (uint32, uint32, bool) { return 0, 0, false }

func Account(string) (uint32, uint32, bool) { return 0, 0, false }

func NoExec(string) bool { return false }

func Writable(string) bool { return true }

func PrivateDir(string) (string, error) { return "", nil }

func PrivatePath(string) string { return "" }

func IsPrivate(string) bool { return false }

func RecordInstance(string, string, string) error { return nil }

func CheckInstance(string) error { return nil }

func RecordedExecutable(running string) string { return running }

const NoFollow = 0
