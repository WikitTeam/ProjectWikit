package pgbundle

import "syscall"

func setDeathSignal(attr *syscall.SysProcAttr) {
	attr.Pdeathsig = syscall.SIGINT
}
