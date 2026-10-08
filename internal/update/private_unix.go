//go:build unix

package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func PrivatePath(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(privateBase(), hex.EncodeToString(sum[:8]))
}

func IsPrivate(path string) bool {
	return strings.HasPrefix(path, privateBase()+string(filepath.Separator))
}

func PrivateDir(root string) (string, error) {
	if os.Geteuid() != 0 {
		return "", nil
	}
	dir := PrivatePath(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s for the programs root runs: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		return "", err
	}
	for d := dir; ; d = filepath.Dir(d) {
		if err := rootOnly(d); err != nil {
			return "", err
		}
		if d == filepath.Dir(d) {
			break
		}
	}
	return dir, nil
}

func privateBase() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/pwikit"
	}
	return "/var/lib/pwikit"
}

func rootOnly(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if info, err = os.Stat(dir); err != nil {
			return err
		}
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%s can be changed by accounts other than root, so root cannot keep its own copy of pwikit there", dir)
	}
	return nil
}
