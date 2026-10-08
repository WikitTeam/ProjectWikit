//go:build unix

package logfile

import "syscall"

const noFollow = syscall.O_NOFOLLOW
