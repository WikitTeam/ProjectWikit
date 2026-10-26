package db

import (
	"context"
	"testing"
	"time"
)

func heldAddress() string {
	return "probe-hold-" + time.Now().Format("150405.000000") + "@example.invalid"
}

func TestSettleEmailHoldsAppliesAHoldThatRanOut(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Hold Due")
	email := heldAddress()
	now := time.Now().UTC()

	if err := d.HoldEmail(ctx, id, email, seedSiteID(t, d)); err != nil {
		t.Fatalf("HoldEmail() err = %v, want nil", err)
	}
	if ok, err := d.ConfirmEmailHold(ctx, id, email, now.Add(-time.Minute)); err != nil || !ok {
		t.Fatalf("ConfirmEmailHold() = %t, %v, want true, nil", ok, err)
	}
	settled, err := d.SettleEmailHolds(ctx, now)
	if err != nil {
		t.Fatalf("SettleEmailHolds() err = %v, want nil", err)
	}
	found := false
	for _, one := range settled {
		if one.UserID == id {
			found = true
			if one.Taken {
				t.Errorf("SettleEmailHolds()[%d].Taken = true, want false", id)
			}
		}
	}
	if !found {
		t.Errorf("SettleEmailHolds() has no %d, want it", id)
	}
	state, err := d.AccountEmail(ctx, id)
	if err != nil {
		t.Fatalf("AccountEmail() err = %v, want nil", err)
	}
	if state.Email != email {
		t.Errorf("AccountEmail().Email = %q, want %q", state.Email, email)
	}
	if state.Held != "" {
		t.Errorf("AccountEmail().Held = %q, want empty", state.Held)
	}
	if state.VerifiedAt == nil {
		t.Errorf("AccountEmail().VerifiedAt = nil, want a time")
	}
}

func TestSettleEmailHoldsLeavesAnUnconfirmedHold(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Hold Waiting")
	email := heldAddress()

	if err := d.HoldEmail(ctx, id, email, seedSiteID(t, d)); err != nil {
		t.Fatalf("HoldEmail() err = %v, want nil", err)
	}
	if _, err := d.SettleEmailHolds(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("SettleEmailHolds() err = %v, want nil", err)
	}
	state, err := d.AccountEmail(ctx, id)
	if err != nil {
		t.Fatalf("AccountEmail() err = %v, want nil", err)
	}
	if state.Held != email {
		t.Errorf("AccountEmail().Held = %q, want %q", state.Held, email)
	}
	if state.Email == email {
		t.Errorf("AccountEmail().Email = %q, want the old address", state.Email)
	}
}

func TestSettleEmailHoldsLeavesAHoldStillRunning(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Hold Running")
	email := heldAddress()
	now := time.Now().UTC()

	if err := d.HoldEmail(ctx, id, email, seedSiteID(t, d)); err != nil {
		t.Fatalf("HoldEmail() err = %v, want nil", err)
	}
	if ok, err := d.ConfirmEmailHold(ctx, id, email, now.Add(time.Hour)); err != nil || !ok {
		t.Fatalf("ConfirmEmailHold() = %t, %v, want true, nil", ok, err)
	}
	if _, err := d.SettleEmailHolds(ctx, now); err != nil {
		t.Fatalf("SettleEmailHolds() err = %v, want nil", err)
	}
	state, err := d.AccountEmail(ctx, id)
	if err != nil {
		t.Fatalf("AccountEmail() err = %v, want nil", err)
	}
	if state.Email == email {
		t.Errorf("AccountEmail().Email = %q, want the old address", state.Email)
	}
}

func TestDropEmailHoldClearsTheHold(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Hold Dropped")
	email := heldAddress()

	if err := d.HoldEmail(ctx, id, email, seedSiteID(t, d)); err != nil {
		t.Fatalf("HoldEmail() err = %v, want nil", err)
	}
	if ok, err := d.DropEmailHold(ctx, id, email); err != nil || !ok {
		t.Fatalf("DropEmailHold() = %t, %v, want true, nil", ok, err)
	}
	state, err := d.AccountEmail(ctx, id)
	if err != nil {
		t.Fatalf("AccountEmail() err = %v, want nil", err)
	}
	if state.Held != "" || state.HeldUntil != nil || state.HeldSite != nil {
		t.Errorf("AccountEmail() hold = %q, %v, %v, want empty", state.Held, state.HeldUntil, state.HeldSite)
	}
}
