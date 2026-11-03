package userpage

import (
	"net/http"
	"time"
)

const (
	fromParam = "from"
	toParam   = "to"
)

// A zero limit means the tab's own pagination applies.
type window struct {
	from, before *time.Time
	site         *int64
	limit        int
}

// The dates are whole days in the site's zone, and the end day is included.
func windowOf(r *http.Request, zone *time.Location) (w window, from, to string) {
	q := r.URL.Query()
	if day, err := time.ParseInLocation(time.DateOnly, q.Get(fromParam), zone); err == nil {
		w.from = &day
		from = day.Format(time.DateOnly)
	}
	if day, err := time.ParseInLocation(time.DateOnly, q.Get(toParam), zone); err == nil {
		next := day.AddDate(0, 0, 1)
		w.before = &next
		to = day.Format(time.DateOnly)
	}
	return w, from, to
}
