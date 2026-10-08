//go:build unix && !linux && !darwin

package update

func noExec(string) bool { return false }
