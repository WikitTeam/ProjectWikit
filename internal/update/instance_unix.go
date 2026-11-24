//go:build unix

package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

func dirIdentity(root string) (uint64, uint64, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return 0, 0, err
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("%s is not a directory", root)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("cannot read the identity of %s", root)
	}
	return uint64(st.Dev), uint64(st.Ino), nil
}

func RecordInstance(private, root, exe string) error {
	exe = ownExecutable(root, exe)
	dev, ino, err := dirIdentity(root)
	if err != nil {
		return err
	}
	data, err := json.Marshal(Instance{Root: root, Executable: exe, Dev: dev, Ino: ino})
	if err != nil {
		return err
	}
	if err := writePrivate(filepath.Join(private, instanceFile), data); err != nil {
		return err
	}
	bin := filepath.Join(private, PrivateBin)
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		return err
	}
	return writePrivate(filepath.Join(bin, paths.DataDirFile), []byte(root+"\n"))
}

func writePrivate(path string, data []byte) error {
	tmp := path + ".part"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (in Instance) Check() error {
	dev, ino, err := dirIdentity(in.Root)
	if os.IsNotExist(err) {
		return fmt.Errorf("the data directory %s no longer exists. If pwikit was moved, run pwikit service uninstall, "+
			"then sudo pwikit service install from its new location", in.Root)
	}
	if err != nil || dev != in.Dev || ino != in.Ino {
		return fmt.Errorf("%s is no longer the directory pwikit was installed in, so pwikit will not run as root from it. "+
			"If pwikit was moved, run pwikit service uninstall, then sudo pwikit service install from its new location; "+
			"otherwise find out who replaced the directory", in.Root)
	}
	return nil
}

func CheckInstance(root string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	in, ok := ReadInstance(PrivatePath(root))
	if !ok {
		return nil
	}
	return in.Check()
}

func RecordedExecutable(running string) string {
	if !IsPrivate(running) {
		return running
	}
	in, ok := ReadInstance(filepath.Dir(filepath.Dir(running)))
	if !ok {
		return running
	}
	if exe := ownExecutable(in.Root, in.Executable); exe != "" && !IsPrivate(exe) {
		return exe
	}
	return running
}

// Updates replace the program the record names. Naming root's own copy there
// would leave the one in the data directory on whatever release it came with.
func ownExecutable(root, exe string) string {
	if exe != "" && !IsPrivate(exe) {
		return exe
	}
	own := filepath.Join(root, ExecutableName(runtime.GOOS))
	if info, err := os.Stat(own); err == nil && info.Mode().IsRegular() {
		return own
	}
	return exe
}
