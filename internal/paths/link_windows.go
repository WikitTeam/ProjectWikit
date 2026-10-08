//go:build windows

package paths

import "io/fs"

func trustedLink(fs.FileInfo) bool { return true }
