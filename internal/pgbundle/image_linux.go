package pgbundle

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func processImage(pid int) (bool, string) {
	if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
		return false, ""
	}
	target, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return true, ""
	}
	return true, strings.TrimSuffix(target, " (deleted)")
}
