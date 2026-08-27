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

func TestUserByDisplayName(t *testing.T) {
	d := newTestDB(t)

	got, err := d.UserByDisplayName(context.Background(), "Probe WD")
	if err != nil {
		t.Fatalf("UserByDisplayName() err = %v, want nil", err)
	}
	if got.WikidotUsername != "probe-wd-original" {
		t.Errorf("UserByDisplayName(Probe WD).WikidotUsername = %q, want %q", got.WikidotUsername, "probe-wd-original")
	}
}

func TestUserByDisplayNameIgnoresCase(t *testing.T) {
	d := newTestDB(t)

	got, err := d.UserByDisplayName(context.Background(), "pRoBe wD")
	if err != nil {
		t.Fatalf("UserByDisplayName() err = %v, want nil", err)
	}
	if got.DisplayName != "Probe WD" {
		t.Errorf("UserByDisplayName(pRoBe wD).DisplayName = %q, want %q", got.DisplayName, "Probe WD")
	}
}

func TestUserByDisplayNameUnknown(t *testing.T) {
	d := newTestDB(t)

	_, err := d.UserByDisplayName(context.Background(), "No Such Display Name")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("UserByDisplayName(unknown) err = %v, want ErrNotFound", err)
	}
}
