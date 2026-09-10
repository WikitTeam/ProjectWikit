// Package version answers which build this is.
package version

import "runtime/debug"

// Release is stamped at link time. A build that was not stamped falls back to
// whatever the module system knows about it.
var Release = ""

func String() string {
	if Release != "" {
		return Release
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				return short(s.Value)
			}
		}
		if info.Main.Version != "" {
			return info.Main.Version
		}
	}
	return "unknown"
}

func short(revision string) string {
	if len(revision) > 12 {
		return revision[:12]
	}
	return revision
}
