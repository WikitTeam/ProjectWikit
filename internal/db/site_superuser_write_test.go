package db

import (
	"context"
	"testing"
	"time"
)

func TestSetSiteSuperuserReportsChange(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := seedSiteID(t, d)
	id := scratchImportedUser(t, d, "Probe Site Super")
	now := time.Now().UTC()

	steps := []struct {
		on   bool
		want bool
	}{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}
	for _, step := range steps {
		got, err := d.SetSiteSuperuser(ctx, site, id, step.on, id, now)
		if err != nil {
			t.Fatalf("SetSiteSuperuser(%t) err = %v, want nil", step.on, err)
		}
		if got != step.want {
			t.Errorf("SetSiteSuperuser(%t) = %t, want %t", step.on, got, step.want)
		}
		held, err := d.IsSiteSuperuser(ctx, site, id)
		if err != nil {
			t.Fatalf("IsSiteSuperuser() err = %v, want nil", err)
		}
		if held != step.on {
			t.Errorf("IsSiteSuperuser() after SetSiteSuperuser(%t) = %t, want %t", step.on, held, step.on)
		}
	}
}

func TestSiteSuperusersOnlyNamesThisSite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := seedSiteID(t, d)
	held := scratchImportedUser(t, d, "Probe Site Super Held")
	plain := scratchImportedUser(t, d, "Probe Site Super Plain")
	if _, err := d.SetSiteSuperuser(ctx, site, held, true, held, time.Now().UTC()); err != nil {
		t.Fatalf("SetSiteSuperuser() err = %v, want nil", err)
	}

	got, err := d.SiteSuperusers(ctx, site, []int64{held, plain})
	if err != nil {
		t.Fatalf("SiteSuperusers() err = %v, want nil", err)
	}
	if !got[held] {
		t.Errorf("SiteSuperusers()[held] = false, want true")
	}
	if got[plain] {
		t.Errorf("SiteSuperusers()[plain] = true, want false")
	}
	if other, err := d.IsSiteSuperuser(ctx, site+1_000_000, held); err != nil || other {
		t.Errorf("IsSiteSuperuser(other site) = %t, %v, want false, nil", other, err)
	}
}
