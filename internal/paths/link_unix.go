//go:build unix

package paths

import (
	"io/fs"
	"os"
	"syscall"
)

func trustedLink(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && (st.Uid == 0 || int(st.Uid) == os.Geteuid())
}
