package userpage

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	showStatsName = "show_stats"
	showTrendName = "show_trend"
	siteParam     = "site"
	yearParam     = "year"
	activityWeeks = 53
)

func (h *Handler) showsStats(ctx context.Context, userID int64) bool {
	return h.profileSwitch(ctx, userID, showStatsName)
}

// Every profile switch is on until its owner turns it off.
func (h *Handler) profileSwitch(ctx context.Context, userID int64, name string) bool {
	raw, err := h.deps.DB.UserPreference(ctx, userID, profileSection, name)
	return err != nil || raw != "False"
}

// The numbers are read through the viewer's permissions, the same as the
// history, so a page they may not see adds nothing to them.
func (h *Handler) activity(r *http.Request, loc *i18n.Localizer, current *db.Site, profile *db.Profile,
	viewer *db.User, data *shell.Profile) error {

	ctx := r.Context()
	sites, err := h.siteSet(ctx, current)
	if err != nil {
		return err
	}
	var hidden []string
	for i := range sites.all {
		one := &sites.all[i]
		cats, err := repo.HiddenCategoriesOn(ctx, h.deps.DB, one, viewer)
		if err != nil {
			return err
		}
		for _, c := range cats {
			hidden = append(hidden, strconv.FormatInt(one.ID, 10)+":"+c)
		}
	}

	wanted := r.URL.Query().Get(siteParam)
	var scope *int64
	data.Sites = append(data.Sites, shell.SiteChoice{Title: loc.T("profile.all-sites"), Picked: wanted == ""})
	for i := range sites.all {
		one := &sites.all[i]
		picked := one.Slug == wanted
		if picked {
			scope = &one.ID
		}
		data.Sites = append(data.Sites, shell.SiteChoice{Slug: one.Slug, Title: one.Title, Picked: picked})
	}

	zone := site.Zone(ctx)
	today := time.Now().In(zone)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, zone)
	// The last year runs up to today. A picked year runs January to December,
	// cut at today while it is still under way.
	end := today
	first := today.AddDate(0, 0, -7*(activityWeeks-1)-int(today.Weekday()))
	year, _ := strconv.Atoi(r.URL.Query().Get(yearParam))
	joined := profile.DateJoined.In(zone).Year()
	if year < joined || year > today.Year() {
		year = 0
	}
	data.Years = append(data.Years, shell.YearChoice{Label: loc.T("profile.last-year"), Picked: year == 0})
	for y := today.Year(); y >= joined; y-- {
		data.Years = append(data.Years, shell.YearChoice{Value: strconv.Itoa(y), Label: strconv.Itoa(y), Picked: y == year})
	}
	if year != 0 {
		jan := time.Date(year, 1, 1, 0, 0, 0, 0, zone)
		first = jan.AddDate(0, 0, -int(jan.Weekday()))
		if dec := time.Date(year, 12, 31, 0, 0, 0, 0, zone); dec.Before(today) {
			end = dec
		}
	}
	days, err := h.deps.DB.ActivityDays(ctx, profile.ID, scope, first, zone.String(), hidden)
	if err != nil {
		return err
	}
	data.Activity = heatmap(loc, days, first, end, year)

	stats, err := h.deps.DB.AuthorStats(ctx, profile.ID, scope, hidden)
	if err != nil {
		return err
	}
	data.Stats = &shell.ProfileStats{Pages: stats.Pages, Score: signed(stats.Score), Edits: stats.Edits, Posts: stats.Posts}

	// Unlike the other switches, the trend stays off until its owner asks for it.
	if raw, err := h.deps.DB.UserPreference(ctx, profile.ID, profileSection, showTrendName); err != nil || raw != "True" {
		return nil
	}
	points, err := h.deps.DB.AuthorScoreTrend(ctx, profile.ID, scope, hidden)
	if err != nil {
		return err
	}
	data.Trend = scoreTrend(points, zone)
	return nil
}

// Levels split the busiest day into four, so a quiet member's best day still
// shows as dark as a busy member's.
func heatmap(loc *i18n.Localizer, days []db.ActivityDay, start, today time.Time, year int) *shell.ProfileActivity {
	counts := make(map[string]int, len(days))
	busiest, total := 0, 0
	for _, d := range days {
		if d.Day.After(today) || (year != 0 && d.Day.Year() != year) {
			continue
		}
		key := d.Day.Format(time.DateOnly)
		counts[key] = d.Count
		busiest = max(busiest, d.Count)
		total += d.Count
	}
	out := &shell.ProfileActivity{Total: total, From: start.Format(time.DateOnly), To: today.Format(time.DateOnly)}
	out.Summary = loc.T("profile.activity-total", "count", total)
	if year != 0 {
		out.Summary = loc.T("profile.activity-year", "year", year, "count", total)
	}
	weeks := int(today.Sub(start).Hours()/24)/7 + 1
	lastMonth := -1
	for w := 0; w < weeks; w++ {
		week := make([]shell.ActivityCell, 7)
		for d := 0; d < 7; d++ {
			day := start.AddDate(0, 0, w*7+d)
			if day.After(today) || (year != 0 && day.Year() != year) {
				week[d] = shell.ActivityCell{Blank: true}
				continue
			}
			key := day.Format(time.DateOnly)
			n := counts[key]
			level := 0
			if n > 0 && busiest > 0 {
				level = int(math.Ceil(float64(n) / float64(busiest) * 4))
			}
			week[d] = shell.ActivityCell{Date: key, Count: n, Level: min(level, 4)}
			if d == 0 && int(day.Month()) != lastMonth {
				if lastMonth != -1 || day.Day() <= 7 {
					out.Months = append(out.Months, shell.ActivityMonth{Column: w, Label: loc.T("profile.month-" + strconv.Itoa(int(day.Month())))})
				}
				lastMonth = int(day.Month())
			}
		}
		out.Weeks = append(out.Weeks, week)
	}
	return out
}

const (
	trendW   = 600.0
	trendH   = 120.0
	trendTop = 8.0
	trendBot = 20.0
	trendL   = 4.0
	trendR   = 44.0
)

func scoreTrend(points []db.TrendPoint, zone *time.Location) *shell.ScoreTrend {
	if len(points) < 2 {
		return nil
	}
	now := time.Now()
	last := points[len(points)-1]
	if now.After(last.At) {
		points = append(points, db.TrendPoint{At: now, Value: last.Value})
	}
	t0, t1 := points[0].At, points[len(points)-1].At
	lo, hi := 0.0, 0.0
	for _, p := range points {
		lo, hi = math.Min(lo, p.Value), math.Max(hi, p.Value)
	}
	if hi == lo {
		hi = lo + 1
	}
	span := math.Max(float64(t1.Sub(t0)), 1)
	x := func(at time.Time) float64 { return trendL + float64(at.Sub(t0))/span*(trendW-trendL-trendR) }
	y := func(v float64) float64 { return trendTop + (1-(v-lo)/(hi-lo))*(trendH-trendTop-trendBot) }
	var b strings.Builder
	fmt.Fprintf(&b, "M%.1f,%.1f", x(points[0].At), y(points[0].Value))
	for _, p := range points[1:] {
		fmt.Fprintf(&b, " H%.1f V%.1f", x(p.At), y(p.Value))
	}
	return &shell.ScoreTrend{
		Path: b.String(), Zero: y(0), Max: signed(hi), Min: signed(lo),
		From: t0.In(zone).Format(time.DateOnly), To: t1.In(zone).Format(time.DateOnly),
		Current: signed(last.Value),
	}
}

func signed(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if v > 0 {
		return "+" + s
	}
	return s
}
