package db

import (
	"context"
	"net/netip"
	"strconv"
	"testing"
	"time"
)

func scratchSession(t *testing.T, d *DB) string {
	t.Helper()
	key := "probedevice" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := d.SaveSession(context.Background(), key, "x", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("SaveSession() err = %v, want nil", err)
	}
	t.Cleanup(func() { _ = d.DeleteSession(context.Background(), key) })
	return key
}

func deviceOf(t *testing.T, d *DB, userID int64, key string) *Device {
	t.Helper()
	list, err := d.Devices(context.Background(), userID)
	if err != nil {
		t.Fatalf("Devices() err = %v, want nil", err)
	}
	for i := range list {
		if list[i].Key == key {
			return &list[i]
		}
	}
	return nil
}

func TestSeenDeviceListsTheSession(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Device List")
	key := scratchSession(t, d)
	addr := netip.MustParseAddr("192.0.2.7")
	now := time.Now().UTC()

	if err := d.SeenDevice(ctx, key, id, nil, true, &addr, "agent", now); err != nil {
		t.Fatalf("SeenDevice() err = %v, want nil", err)
	}
	got := deviceOf(t, d, id, key)
	if got == nil {
		t.Fatalf("Devices() has no %q, want it", key)
	}
	if got.Address != "192.0.2.7" {
		t.Errorf("Devices().Address = %q, want %q", got.Address, "192.0.2.7")
	}
	if got.SignedInAt == nil {
		t.Errorf("Devices().SignedInAt = nil, want a time")
	}
}

func TestSeenDeviceSkipsAMissingSession(t *testing.T) {
	d := writeTestDB(t)
	id := scratchImportedUser(t, d, "Probe Device Gone")
	if err := d.SeenDevice(context.Background(), "probedevicegone", id, nil, false, nil, "", time.Now()); err != nil {
		t.Fatalf("SeenDevice() err = %v, want nil", err)
	}
	if got := deviceOf(t, d, id, "probedevicegone"); got != nil {
		t.Errorf("Devices() has %q, want none", "probedevicegone")
	}
}

func TestEndOtherDevicesKeepsTheCurrentOne(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Device Others")
	keep := scratchSession(t, d)
	time.Sleep(time.Millisecond)
	other := scratchSession(t, d)
	for _, key := range []string{keep, other} {
		if err := d.SeenDevice(ctx, key, id, nil, true, nil, "", time.Now()); err != nil {
			t.Fatalf("SeenDevice(%q) err = %v, want nil", key, err)
		}
	}

	ended, err := d.EndOtherDevices(ctx, id, keep)
	if err != nil {
		t.Fatalf("EndOtherDevices() err = %v, want nil", err)
	}
	if ended != 1 {
		t.Errorf("EndOtherDevices() = %d, want 1", ended)
	}
	if deviceOf(t, d, id, keep) == nil {
		t.Errorf("Devices() has no %q, want it", keep)
	}
	if _, _, err := d.SessionByKey(ctx, other); err != ErrNotFound {
		t.Errorf("SessionByKey(%q) err = %v, want ErrNotFound", other, err)
	}
}

func TestEndDeviceRefusesAnotherUsersDevice(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	owner := scratchImportedUser(t, d, "Probe Device Owner")
	stranger := scratchImportedUser(t, d, "Probe Device Stranger")
	key := scratchSession(t, d)
	if err := d.SeenDevice(ctx, key, owner, nil, true, nil, "", time.Now()); err != nil {
		t.Fatalf("SeenDevice() err = %v, want nil", err)
	}
	dev := deviceOf(t, d, owner, key)
	if dev == nil {
		t.Fatalf("Devices() has no %q, want it", key)
	}

	if ok, err := d.EndDevice(ctx, stranger, dev.ID); err != nil || ok {
		t.Errorf("EndDevice(stranger) = %t, %v, want false, nil", ok, err)
	}
	if ok, err := d.EndDevice(ctx, owner, dev.ID); err != nil || !ok {
		t.Errorf("EndDevice(owner) = %t, %v, want true, nil", ok, err)
	}
}
