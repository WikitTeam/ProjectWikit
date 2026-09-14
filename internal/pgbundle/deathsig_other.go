//go:build unix && !linux

package pgbundle

import "syscall"

func setDeathSignal(*syscall.SysProcAttr) {}
