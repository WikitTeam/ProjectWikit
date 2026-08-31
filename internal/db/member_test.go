package db

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

func TestMembersAreOrderedByID(t *testing.T) {
	d := newTestDB(t)

	got, err := d.Members(context.Background(), nil, 0, 100)
	if err != nil {
		t.Fatalf("Members() err = %v, want nil", err)
	}
	if len(got) == 0 {
		t.Fatal("Members() = 0 rows, want at least one")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ID >= got[i].ID {
			t.Errorf("Members()[%d].ID = %d, want it after %d", i, got[i].ID, got[i-1].ID)
		}
	}
	if got[0].JoinedAt.IsZero() {
		t.Errorf("Members()[0].JoinedAt = %v, want a real time", got[0].JoinedAt)
	}
}

func TestMemberCountMatchesTheUnfilteredListing(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	total, err := d.MemberCount(ctx, nil)
	if err != nil {
		t.Fatalf("MemberCount() err = %v, want nil", err)
	}
	listed, err := d.Members(ctx, nil, 0, total+1)
	if err != nil {
		t.Fatalf("Members() err = %v, want nil", err)
	}
	if len(listed) != total {
		t.Errorf("MemberCount() = %d, want %d", total, len(listed))
	}
}

func TestMembersOffsetSkipsRows(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	all, err := d.Members(ctx, nil, 0, 3)
	if err != nil {
		t.Fatalf("Members() err = %v, want nil", err)
	}
	if len(all) < 2 {
		t.Skip("the database holds fewer than two users")
	}
	got, err := d.Members(ctx, nil, 1, 1)
	if err != nil {
		t.Fatalf("Members() err = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("Members(offset 1, limit 1) = %d rows, want 1", len(got))
	}
	if got[0].ID != all[1].ID {
		t.Errorf("Members(offset 1).ID = %d, want %d", got[0].ID, all[1].ID)
	}
}

func TestMembersFilteredByRole(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	role, err := d.RoleByRef(ctx, "everyone")
	if err != nil {
		t.Fatalf("RoleByRef(\"everyone\") err = %v, want nil", err)
	}
	filtered, err := d.MemberCount(ctx, &role.ID)
	if err != nil {
		t.Fatalf("MemberCount() err = %v, want nil", err)
	}
	total, err := d.MemberCount(ctx, nil)
	if err != nil {
		t.Fatalf("MemberCount() err = %v, want nil", err)
	}
	if filtered > total {
		t.Errorf("MemberCount(role) = %d, want at most %d", filtered, total)
	}
}

func TestRoleByRefMatchesTheNumber(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	bySlug, err := d.RoleByRef(ctx, "everyone")
	if err != nil {
		t.Fatalf("RoleByRef(\"everyone\") err = %v, want nil", err)
	}
	byID, err := d.RoleByRef(ctx, itoa(bySlug.ID))
	if err != nil {
		t.Fatalf("RoleByRef(%q) err = %v, want nil", itoa(bySlug.ID), err)
	}
	if byID.Slug != bySlug.Slug {
		t.Errorf("RoleByRef(%q).Slug = %q, want %q", itoa(bySlug.ID), byID.Slug, bySlug.Slug)
	}
}

func TestRoleByRefUnknown(t *testing.T) {
	d := newTestDB(t)

	_, err := d.RoleByRef(context.Background(), "no-such-role")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("RoleByRef(\"no-such-role\") err = %v, want ErrNotFound", err)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
