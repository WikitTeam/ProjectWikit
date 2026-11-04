package db

import (
	"context"
	"testing"
	"time"
)

func scratchAuthoredPost(t *testing.T, d *DB, threadID, author int64, at time.Time) {
	t.Helper()
	var id int64
	err := d.pool.QueryRow(context.Background(), `
INSERT INTO web_forumpost (thread_id, author_id, name, created_at, updated_at)
VALUES ($1, $2, 'probe', $3, $3)
RETURNING id`, threadID, author, at).Scan(&id)
	if err != nil {
		t.Fatalf("insert scratch post err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_forumpost WHERE id = $1`, id); err != nil {
			t.Errorf("clean up scratch post err = %v, want nil", err)
		}
	})
}

func TestActivityDaysUsesTheZone(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	thread := scratchThread(t, d)
	user := scratchUser(t, d, "probe-activity")
	for _, at := range []string{
		"2026-03-01T15:50:00Z",
		"2026-03-01T16:10:00Z",
		"2026-03-05T18:14:00Z",
		"2026-03-05T18:16:00Z",
	} {
		when, _ := time.Parse(time.RFC3339, at)
		scratchAuthoredPost(t, d, thread, user, when)
	}
	since := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		zone string
		want map[string]int
	}{
		{"UTC", map[string]int{"2026-03-01": 2, "2026-03-05": 2}},
		{"Asia/Shanghai", map[string]int{"2026-03-01": 1, "2026-03-02": 1, "2026-03-06": 2}},
		{"Asia/Kathmandu", map[string]int{"2026-03-01": 2, "2026-03-05": 1, "2026-03-06": 1}},
	}
	for _, c := range cases {
		t.Run(c.zone, func(t *testing.T) {
			zone, err := time.LoadLocation(c.zone)
			if err != nil {
				t.Fatalf("LoadLocation(%q) err = %v, want nil", c.zone, err)
			}
			days, err := d.ActivityDays(ctx, user, nil, since, zone, nil)
			if err != nil {
				t.Fatalf("ActivityDays() err = %v, want nil", err)
			}
			got := map[string]int{}
			for _, day := range days {
				got[day.Day.Format(time.DateOnly)] = day.Count
			}
			if len(got) != len(c.want) {
				t.Errorf("ActivityDays() = %v, want %v", got, c.want)
			}
			for day, n := range c.want {
				if got[day] != n {
					t.Errorf("ActivityDays()[%s] = %d, want %d", day, got[day], n)
				}
			}
		})
	}
}
