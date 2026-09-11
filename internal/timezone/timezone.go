// Package timezone answers which zone a site's dates fall in on the server.
package timezone

import (
	"sync"
	"time"

	// Windows and scratch containers carry no zone database.
	_ "time/tzdata"
)

var loaded sync.Map

func Load(name string) *time.Location {
	if cached, ok := loaded.Load(name); ok {
		return cached.(*time.Location)
	}
	if !Valid(name) {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	loaded.Store(name, loc)
	return loc
}

// Local is the server machine's zone, which a site's zone must never depend on.
func Valid(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

func Label(at time.Time) string {
	_, offset := at.Zone()
	if offset == 0 {
		return "UTC"
	}
	return "UTC" + at.Format("-07:00")
}
