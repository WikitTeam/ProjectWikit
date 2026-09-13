//go:build windows

package shellpath

import (
	"errors"
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	environmentKey = `Environment`
	markerKey      = `Software\ProjectWikit`
	markerValue    = "PathEntry"
)

func readUserPath() (string, uint32, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, environmentKey, registry.QUERY_VALUE)
	if err != nil {
		return "", registry.EXPAND_SZ, err
	}
	defer key.Close()
	value, kind, err := key.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return "", registry.EXPAND_SZ, nil
	}
	return value, kind, err
}

func writeUserPath(value string, kind uint32) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, environmentKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if kind == registry.SZ {
		return key.SetStringValue("Path", value)
	}
	return key.SetExpandStringValue("Path", value)
}

// Path cannot say which of its entries pwikit added once the program has moved,
// so the entry is remembered beside it.
func readMarker() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, markerKey, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	value, _, err := key.GetStringValue(markerValue)
	if err != nil {
		return ""
	}
	return value
}

func writeMarker(dir string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, markerKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(markerValue, dir)
}

func clearMarker() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, markerKey, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer key.Close()
	if err := key.DeleteValue(markerValue); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

// Programs started after this pick up the new Path. Terminals already open keep
// the one they started with.
func announce() {
	send := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
	name, err := windows.UTF16PtrFromString(environmentKey)
	if err != nil {
		return
	}
	const (
		broadcast     = 0xffff
		settingChange = 0x001a
		abortIfHung   = 0x0002
	)
	var result uintptr
	send.Call(broadcast, settingChange, 0, uintptr(unsafe.Pointer(name)), abortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}

func place(dir, exe, value string) Place {
	target := filepath.Join(dir, command+".exe")
	return Place{Path: dir, Target: target, Working: exists(target), Ours: sameFile(target, exe), OnPath: pathHas(value, dir)}
}

func Install(exe, dir string, force bool) (Place, Outcome, error) {
	if dir != "" {
		return Place{}, "", errors.New("on Windows pwikit adds the directory holding it to your Path, so -dir does not apply")
	}
	here := filepath.Dir(exe)
	value, kind, err := readUserPath()
	if err != nil {
		return Place{}, "", fmt.Errorf("read your Path: %w", err)
	}
	marker := readMarker()

	if pathHas(value, here) && sameEntry(marker, here) {
		return place(here, exe, value), Unchanged, nil
	}
	if !force {
		for _, part := range splitPath(value) {
			if sameEntry(part, here) || sameEntry(part, marker) {
				continue
			}
			if exists(filepath.Join(part, command+".exe")) {
				return Place{}, "", fmt.Errorf("%s is already on your Path and holds another pwikit; pass -force to use %s instead", part, here)
			}
		}
	}

	outcome := Created
	next := value
	if marker != "" && !sameEntry(marker, here) {
		next = pathWithout(next, marker)
		outcome = Replaced
	}
	next = pathWith(next, here)
	if next != value {
		if err := writeUserPath(next, kind); err != nil {
			return Place{}, "", fmt.Errorf("write your Path: %w", err)
		}
	}
	if err := writeMarker(here); err != nil {
		return Place{}, "", err
	}
	announce()
	return place(here, exe, next), outcome, nil
}

func Uninstall(exe, dir string) (Place, Outcome, error) {
	value, kind, err := readUserPath()
	if err != nil {
		return Place{}, "", fmt.Errorf("read your Path: %w", err)
	}
	marker := readMarker()
	if marker == "" {
		return Place{}, Absent, nil
	}
	p := place(marker, exe, value)
	if pathHas(value, marker) {
		if err := writeUserPath(pathWithout(value, marker), kind); err != nil {
			return p, "", fmt.Errorf("write your Path: %w", err)
		}
	}
	if err := clearMarker(); err != nil {
		return p, "", err
	}
	announce()
	return p, Removed, nil
}

func Status(exe, dir string) ([]Place, error) {
	value, _, err := readUserPath()
	if err != nil {
		return nil, fmt.Errorf("read your Path: %w", err)
	}
	var out []Place
	if marker := readMarker(); marker != "" {
		out = append(out, place(marker, exe, value))
	}
	return out, nil
}
