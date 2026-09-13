//go:build !windows

package shellpath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// With no directory named, the system-wide one is tried first and the account's
// own is the one that works without root.
func candidates(dir string) []string {
	if dir != "" {
		return []string{dir}
	}
	out := []string{"/usr/local/bin"}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".local", "bin"))
	}
	return out
}

func inspect(link string) entry {
	info, err := os.Lstat(link)
	if err != nil {
		return entry{}
	}
	e := entry{present: true, link: info.Mode()&os.ModeSymlink != 0}
	if !e.link {
		e.working = true
		return e
	}
	target, err := os.Readlink(link)
	if err != nil {
		return e
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	e.target = target
	e.working = exists(target)
	return e
}

func place(link, exe string) Place {
	e := inspect(link)
	p := Place{Path: link, Target: e.target, Working: e.working, OnPath: onPath(filepath.Dir(link))}
	p.Ours = e.link && sameFile(e.target, exe)
	return p
}

func Install(exe, dir string, force bool) (Place, Outcome, error) {
	for _, candidate := range candidates(dir) {
		link := filepath.Join(candidate, command)
		if e := inspect(link); e.link && sameFile(e.target, exe) {
			return place(link, exe), Unchanged, nil
		}
	}

	var lastErr error
	for _, candidate := range candidates(dir) {
		link := filepath.Join(candidate, command)
		existing := inspect(link)
		if decide(existing, exe, force) == actRefuse {
			if existing.link {
				return Place{}, "", fmt.Errorf("%s already leads to %s; pass -force to point it at %s", link, existing.target, exe)
			}
			return Place{}, "", fmt.Errorf("%s already exists and is not a link; pass -force to replace it", link)
		}
		outcome := Created
		if existing.present {
			outcome = Replaced
		}
		if err := os.MkdirAll(candidate, 0o755); err != nil {
			lastErr = err
			continue
		}
		if err := replaceLink(link, exe); err != nil {
			lastErr = err
			if dir == "" && errors.Is(err, os.ErrPermission) {
				continue
			}
			return Place{}, "", err
		}
		return place(link, exe), outcome, nil
	}
	return Place{}, "", fmt.Errorf("no directory could take the link: %w", lastErr)
}

// The new link is made beside the old name and renamed over it, so a shell
// never finds the command missing halfway through.
func replaceLink(link, exe string) error {
	temp := link + ".pwikit-new"
	os.Remove(temp)
	if err := os.Symlink(exe, temp); err != nil {
		return err
	}
	if err := os.Rename(temp, link); err != nil {
		os.Remove(temp)
		return err
	}
	return nil
}

func Uninstall(exe, dir string) (Place, Outcome, error) {
	for _, candidate := range candidates(dir) {
		link := filepath.Join(candidate, command)
		e := inspect(link)
		if !e.link || (e.working && !sameFile(e.target, exe)) {
			continue
		}
		p := place(link, exe)
		if err := os.Remove(link); err != nil {
			return p, "", err
		}
		return p, Removed, nil
	}
	return Place{}, Absent, nil
}

func Status(exe, dir string) ([]Place, error) {
	var out []Place
	for _, candidate := range candidates(dir) {
		link := filepath.Join(candidate, command)
		if inspect(link).present {
			out = append(out, place(link, exe))
		}
	}
	return out, nil
}
