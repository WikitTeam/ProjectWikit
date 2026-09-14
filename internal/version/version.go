// Package version answers which build this is.
package version

import (
	"runtime/debug"
	"time"
)

// Release is stamped at link time. A build that was not stamped falls back to
// whatever the module system knows about it.
var Release = ""

func String() string {
	if Release != "" {
		return Release
	}
	if revision := setting("vcs.revision"); revision != "" {
		return short(revision)
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "unknown"
}

func Commit() string {
	revision := short(setting("vcs.revision"))
	if revision != "" && setting("vcs.modified") == "true" {
		revision += "-modified"
	}
	return revision
}

func BuiltFrom() time.Time {
	at, err := time.Parse(time.RFC3339, setting("vcs.time"))
	if err != nil {
		return time.Time{}
	}
	return at
}

func setting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

func short(revision string) string {
	if len(revision) > 12 {
		return revision[:12]
	}
	return revision
}
