package db

import (
	"context"
	"errors"
	"testing"
)

func TestSiteByHostsMatchesDomainAndMediaDomain(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	byDomain, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts([localhost]) err = %v, want nil", err)
	}
	if byDomain.Slug != "wikit" {
		t.Errorf("SiteByHosts([localhost]).Slug = %q, want %q", byDomain.Slug, "wikit")
	}

	byMedia, err := d.SiteByHosts(ctx, []string{"media.localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts([media.localhost]) err = %v, want nil", err)
	}
	if byMedia.ID != byDomain.ID {
		t.Errorf("SiteByHosts([media.localhost]).ID = %d, want %d", byMedia.ID, byDomain.ID)
	}
}

func TestSiteByHostsTriesHostsInOrder(t *testing.T) {
	d := newTestDB(t)

	got, err := d.SiteByHosts(context.Background(), []string{"localhost:8000", "localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts() err = %v, want nil", err)
	}
	if got.Domain != "localhost" {
		t.Errorf("SiteByHosts().Domain = %q, want %q", got.Domain, "localhost")
	}
}

func TestSiteByHostsUnknownHost(t *testing.T) {
	d := newTestDB(t)

	_, err := d.SiteByHosts(context.Background(), []string{"nope.example"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("SiteByHosts([nope.example]) err = %v, want ErrNotFound", err)
	}
}

func TestSiteHostExistsAcceptsBothDomainsAndRejectsStrangers(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	for _, host := range []string{"localhost", "media.localhost", "LOCALHOST"} {
		got, err := d.SiteHostExists(ctx, host)
		if err != nil {
			t.Fatalf("SiteHostExists(%q) err = %v, want nil", host, err)
		}
		if !got {
			t.Errorf("SiteHostExists(%q) = false, want true", host)
		}
	}

	got, err := d.SiteHostExists(ctx, "someone-elses-domain.example")
	if err != nil {
		t.Fatalf("SiteHostExists() err = %v, want nil", err)
	}
	if got {
		t.Error("SiteHostExists(\"someone-elses-domain.example\") = true, want false")
	}
}

func seedSiteID(t *testing.T, d *DB) int64 {
	t.Helper()
	found, err := d.SiteByHosts(context.Background(), []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}
	return found.ID
}
