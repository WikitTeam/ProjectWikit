package db

import (
	"context"
	"testing"
)

func TestWantedLinkSQLMatchesSchema(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	for name, filter := range map[string]WantedFilter{
		"empty":          {},
		"from":           {From: []string{"probestars:unrated"}},
		"categories":     {From: []string{"probestars:unrated"}, Categories: []string{"wanted"}},
		"not-categories": {From: []string{"probestars:unrated"}, NotCategories: []string{"wanted"}},
		"both":           {From: []string{"probestars:unrated"}, Categories: []string{"wanted"}, NotCategories: []string{"probe"}},
	} {
		if _, err := d.WantedLinks(ctx, filter, 0, 20); err != nil {
			t.Errorf("WantedLinks(%s) err = %v, want nil", name, err)
		}
		if _, err := d.WantedLinkCount(ctx, filter); err != nil {
			t.Errorf("WantedLinkCount(%s) err = %v, want nil", name, err)
		}
	}
}

func TestWantedLinkCountAgreesWithTheRows(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	filter := WantedFilter{From: []string{"probestars:unrated", "probestars:quarter", "probeoff:unratable"}}

	rows, err := d.WantedLinks(ctx, filter, 0, 100)
	if err != nil {
		t.Fatalf("WantedLinks() err = %v, want nil", err)
	}
	total, err := d.WantedLinkCount(ctx, filter)
	if err != nil {
		t.Fatalf("WantedLinkCount() err = %v, want nil", err)
	}
	if total != len(rows) {
		t.Errorf("WantedLinkCount() = %d, want %d", total, len(rows))
	}
}

func TestWantedLinksSkipPagesThatExist(t *testing.T) {
	d := newTestDB(t)
	filter := WantedFilter{From: []string{"probeoff:unratable"}}

	rows, err := d.WantedLinks(context.Background(), filter, 0, 100)
	if err != nil {
		t.Fatalf("WantedLinks() err = %v, want nil", err)
	}
	for _, row := range rows {
		if row.To == "probe:full" {
			t.Errorf("WantedLinks() has %q, want only names with no page", row.To)
		}
	}
}
