//go:build !linux && !darwin && !windows

package service

import (
	"errors"
	"os/user"
)

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

func Running(string) (bool, error) { return false, errUnsupported }

func Lookup(string) (Installed, bool) { return Installed{}, false }

func PointUpdateAt(string, string) (bool, error) { return false, nil }

func PointServiceAt(string, string) (bool, error) { return false, nil }

func ServiceUser(string) (*user.User, error) { return nil, errUnsupported }
