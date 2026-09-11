//go:build !windows

package main

func runAsService([]string) (bool, error) {
	return false, nil
}
