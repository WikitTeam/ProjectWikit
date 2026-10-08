package update

import "golang.org/x/sys/unix"

func noExec(dir string) bool {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return false
	}
	return st.Flags&unix.ST_NOEXEC != 0
}
