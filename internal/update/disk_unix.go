//go:build unix

package update

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func FreeBytes(dir string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}

var ErrCannotHandOver = errors.New("the filesystem does not let root change who owns a file")

func ChownTree(path, root string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	uid, gid, ok := Account(root)
	if !ok {
		return nil
	}
	var refused error
	err := filepath.WalkDir(path, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if ok && st.Uid == uid && st.Gid == gid {
			return nil
		}
		if ok && !info.IsDir() && st.Nlink > 1 {
			return nil
		}
		if err := os.Lchown(p, int(uid), int(gid)); err != nil {
			if !chownRefused(err) {
				return err
			}
			if refused == nil {
				refused = fmt.Errorf("%w: %s: %v", ErrCannotHandOver, p, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return refused
}

func chownRefused(err error) bool {
	for _, errno := range []error{syscall.EPERM, syscall.EINVAL, syscall.EROFS, syscall.ENOTSUP} {
		if errors.Is(err, errno) {
			return true
		}
	}
	return false
}

func Owner(path string) (uint32, uint32, bool) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, 0, false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return st.Uid, st.Gid, true
}

func Account(root string) (uint32, uint32, bool) {
	uid, gid, ok := Owner(root)
	if !ok {
		return 0, 0, false
	}
	if uid != 0 {
		return uid, gid, true
	}
	for _, name := range stateNames {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if uid, gid, ok := Owner(path); ok && uid != 0 {
			return uid, gid, true
		}
	}
	return 0, 0, true
}

var stateNames = []string{"pgdata", "update", "logs", "files", "archive", "backups"}

func NoExec(dir string) bool {
	return noExec(dir)
}

func Writable(dir string) bool {
	return unix.Access(dir, unix.W_OK) == nil
}

const NoFollow = syscall.O_NOFOLLOW
