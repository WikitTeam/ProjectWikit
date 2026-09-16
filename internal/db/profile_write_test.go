package db

import (
	"context"
	"testing"
	"time"
)

func TestRolesOfUserOnEverySiteGroupsBySite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	here, there := scratchSite(t, d), scratchSite(t, d)
	roleHere, roleThere := scratchRole(t, d, here), scratchRole(t, d, there)

	userID, err := d.CreateUser(ctx, scratchName(t), "Probe Roles", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, userID)
	for site, role := range map[int64]int64{here: roleHere, there: roleThere} {
		if err := d.GrantRole(ctx, site, userID, role); err != nil {
			t.Fatalf("GrantRole(%d) err = %v, want nil", role, err)
		}
	}

	got, err := d.RolesOfUserOnEverySite(ctx, userID)
	if err != nil {
		t.Fatalf("RolesOfUserOnEverySite() err = %v, want nil", err)
	}
	bySite := map[int64][]int64{}
	for _, one := range got {
		for _, role := range one.Roles {
			bySite[one.SiteID] = append(bySite[one.SiteID], role.ID)
		}
	}
	if len(got) != 2 {
		t.Fatalf("len(RolesOfUserOnEverySite()) = %d, want 2", len(got))
	}
	if ids := bySite[here]; len(ids) != 1 || ids[0] != roleHere {
		t.Errorf("RolesOfUserOnEverySite()[here] = %v, want [%d]", ids, roleHere)
	}
	if ids := bySite[there]; len(ids) != 1 || ids[0] != roleThere {
		t.Errorf("RolesOfUserOnEverySite()[there] = %v, want [%d]", ids, roleThere)
	}
}

func TestUserPostsCountsCommentsOnlyOnTheNamedSites(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	seed := seedSiteID(t, d)
	other := scratchSite(t, d)
	thread := scratchThread(t, d)
	if _, err := d.pool.Exec(ctx, `UPDATE web_forumthread SET site_id = $1 WHERE id = $2`, seed, thread); err != nil {
		t.Fatalf("set thread site err = %v, want nil", err)
	}
	user := scratchUser(t, d, "probe-comment-author")
	post := scratchPost(t, d, thread, "probe")
	if _, err := d.pool.Exec(ctx, `UPDATE web_forumpost SET author_id = $1 WHERE id = $2`, user, post); err != nil {
		t.Fatalf("set post author err = %v, want nil", err)
	}

	cases := []struct {
		name  string
		sites []int64
		want  int
	}{
		{"its own site", []int64{seed}, 1},
		{"another site", []int64{other}, 0},
		{"no site", nil, 0},
	}
	for _, c := range cases {
		got, err := d.UserPostCount(ctx, user, nil, c.sites)
		if err != nil {
			t.Fatalf("UserPostCount(%s) err = %v, want nil", c.name, err)
		}
		if got != c.want {
			t.Errorf("UserPostCount(%s) = %d, want %d", c.name, got, c.want)
		}
		posts, err := d.UserPosts(ctx, user, nil, c.sites, 0, 10)
		if err != nil {
			t.Fatalf("UserPosts(%s) err = %v, want nil", c.name, err)
		}
		if len(posts) != c.want {
			t.Errorf("len(UserPosts(%s)) = %d, want %d", c.name, len(posts), c.want)
		}
	}
}

func TestRecentPostsCountsCommentsOnlyOnTheSite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	seed := seedSiteID(t, d)
	other := scratchSite(t, d)
	thread := scratchThread(t, d)
	if _, err := d.pool.Exec(ctx, `UPDATE web_forumthread SET site_id = $1 WHERE id = $2`, seed, thread); err != nil {
		t.Fatalf("set thread site err = %v, want nil", err)
	}
	scratchPost(t, d, thread, "probe")

	here, err := d.RecentPostCount(ctx, seed, nil, true)
	if err != nil {
		t.Fatalf("RecentPostCount(seed) err = %v, want nil", err)
	}
	there, err := d.RecentPostCount(ctx, other, nil, true)
	if err != nil {
		t.Fatalf("RecentPostCount(other) err = %v, want nil", err)
	}
	if there != 0 {
		t.Errorf("RecentPostCount(other) = %d, want 0", there)
	}
	if here < 1 {
		t.Errorf("RecentPostCount(seed) = %d, want at least 1", here)
	}
}
