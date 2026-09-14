//go:build windows

package update

import "golang.org/x/sys/windows"

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

func ChownTree(string, string) error { return nil }

func Owner(string) (uint32, uint32, bool) { return 0, 0, false }
