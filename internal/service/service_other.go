//go:build !linux && !darwin && !windows

package service

import "errors"

var errUnsupported = errors.New("pwikit can only register itself on Linux with systemd, macOS and Windows.\n" +
	"  Have your init system run the command that pwikit service print shows")

func Preview(s Spec) (string, error) {
	return s.Systemd(), nil
}

func Install(Spec) error     { return errUnsupported }
func Uninstall(string) error { return errUnsupported }
func Start(string) error     { return errUnsupported }
func Stop(string) error      { return errUnsupported }
func Status(string) error    { return errUnsupported }
