package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSessionRoundTrip(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	key := "gotestsessionkey00000000000000aa"
	t.Cleanup(func() { d.DeleteSession(ctx, key) })

	if err := d.SaveSession(ctx, key, "payload", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("SaveSession() err = %v, want nil", err)
	}
	data, expires, err := d.SessionByKey(ctx, key)
	if err != nil {
		t.Fatalf("SessionByKey() err = %v, want nil", err)
	}
	if data != "payload" {
		t.Errorf("SessionByKey() data = %q, want %q", data, "payload")
	}
	if expires.Before(time.Now()) {
		t.Errorf("SessionByKey() expires = %v, want a future time", expires)
	}
}

func TestSaveSessionOverwrites(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	key := "gotestsessionkey00000000000000bb"
	t.Cleanup(func() { d.DeleteSession(ctx, key) })

	if err := d.SaveSession(ctx, key, "first", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("SaveSession(first) err = %v, want nil", err)
	}
	if err := d.SaveSession(ctx, key, "second", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("SaveSession(second) err = %v, want nil", err)
	}
	data, _, err := d.SessionByKey(ctx, key)
	if err != nil {
		t.Fatalf("SessionByKey() err = %v, want nil", err)
	}
	if data != "second" {
		t.Errorf("SessionByKey() data = %q, want %q", data, "second")
	}
}

func TestSessionByKeyTreatsExpiredAsMissing(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	key := "gotestsessionkey00000000000000cc"
	t.Cleanup(func() { d.DeleteSession(ctx, key) })

	if err := d.SaveSession(ctx, key, "payload", time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("SaveSession() err = %v, want nil", err)
	}
	if _, _, err := d.SessionByKey(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("SessionByKey(expired) err = %v, want ErrNotFound", err)
	}
}

func TestSessionByKeyUnknown(t *testing.T) {
	d := newTestDB(t)

	if _, _, err := d.SessionByKey(context.Background(), "gotestsessionkey0000000000000zz"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SessionByKey(unknown) err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSession(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	key := "gotestsessionkey00000000000000dd"

	if err := d.SaveSession(ctx, key, "payload", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("SaveSession() err = %v, want nil", err)
	}
	if err := d.DeleteSession(ctx, key); err != nil {
		t.Fatalf("DeleteSession() err = %v, want nil", err)
	}
	if _, _, err := d.SessionByKey(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("SessionByKey(deleted) err = %v, want ErrNotFound", err)
	}
}
