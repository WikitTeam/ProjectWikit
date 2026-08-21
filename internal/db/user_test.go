package db

import (
	"context"
	"errors"
	"testing"
)

func TestUserByName(t *testing.T) {
	d := newTestDB(t)

	got, err := d.UserByName(context.Background(), "seeduser")
	if err != nil {
		t.Fatalf("UserByName(\"seeduser\") err = %v, want nil", err)
	}
	if got.Username != "seeduser" {
		t.Errorf("UserByName().Username = %q, want %q", got.Username, "seeduser")
	}
	if got.Type != UserTypeNormal {
		t.Errorf("UserByName().Type = %q, want %q", got.Type, UserTypeNormal)
	}
}

func TestUserByNameUnknown(t *testing.T) {
	d := newTestDB(t)

	_, err := d.UserByName(context.Background(), "no-such-user")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("UserByName(\"no-such-user\") err = %v, want ErrNotFound", err)
	}
}
