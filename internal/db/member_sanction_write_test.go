package db

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestSanctionsOnlyCountWhileTheyLast(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := scratchSite(t, d)
	now := time.Now().UTC()

	userID, err := d.CreateUser(ctx, scratchName(t), "Probe Sanctioned", "!", true, now)
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, userID)

	forever := now.Add(time.Hour)
	if err := d.SetSanction(ctx, site, userID, "ban", nil, "spam", &userID, now); err != nil {
		t.Fatalf("SetSanction(ban) err = %v, want nil", err)
	}
	if err := d.SetSanction(ctx, site, userID, "mute", &forever, "noise", &userID, now); err != nil {
		t.Fatalf("SetSanction(mute) err = %v, want nil", err)
	}
	gone := now.Add(-time.Hour)
	if err := d.SetSanction(ctx, site, userID, "rating", &gone, "over", &userID, now); err != nil {
		t.Fatalf("SetSanction(rating) err = %v, want nil", err)
	}

	kinds, err := d.ActiveSanctions(ctx, site, userID, now)
	if err != nil {
		t.Fatalf("ActiveSanctions() err = %v, want nil", err)
	}
	slices.Sort(kinds)
	if !slices.Equal(kinds, []string{"ban", "mute"}) {
		t.Errorf("ActiveSanctions() = %v, want [ban mute]", kinds)
	}

	stored, err := d.MemberSanctions(ctx, site, userID)
	if err != nil {
		t.Fatalf("MemberSanctions() err = %v, want nil", err)
	}
	if len(stored) != 3 {
		t.Errorf("len(MemberSanctions()) = %d, want 3", len(stored))
	}
	for _, one := range stored {
		if one.Kind == "ban" && one.Reason != "spam" {
			t.Errorf("MemberSanctions()[ban].Reason = %q, want %q", one.Reason, "spam")
		}
	}

	if err := d.ClearSanction(ctx, site, userID, "ban"); err != nil {
		t.Fatalf("ClearSanction() err = %v, want nil", err)
	}
	kinds, err = d.ActiveSanctions(ctx, site, userID, now)
	if err != nil {
		t.Fatalf("ActiveSanctions() err = %v, want nil", err)
	}
	if !slices.Equal(kinds, []string{"mute"}) {
		t.Errorf("ActiveSanctions() after lifting the ban = %v, want [mute]", kinds)
	}
}

func TestSanctionsAreKeptPerSite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	here, there := scratchSite(t, d), scratchSite(t, d)
	now := time.Now().UTC()

	userID, err := d.CreateUser(ctx, scratchName(t), "Probe Elsewhere", "!", true, now)
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, userID)

	if err := d.SetSanction(ctx, here, userID, "ban", nil, "", nil, now); err != nil {
		t.Fatalf("SetSanction() err = %v, want nil", err)
	}
	kinds, err := d.ActiveSanctions(ctx, there, userID, now)
	if err != nil {
		t.Fatalf("ActiveSanctions() err = %v, want nil", err)
	}
	if len(kinds) != 0 {
		t.Errorf("ActiveSanctions(other site) = %v, want none", kinds)
	}
}
