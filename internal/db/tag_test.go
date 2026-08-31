package db

import (
	"context"
	"testing"
)

func TestTagCloudSQLMatchesSchema(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	limit := 2

	all, err := d.TagCloud(ctx, nil)
	if err != nil {
		t.Fatalf("TagCloud(nil) err = %v, want nil", err)
	}
	limited, err := d.TagCloud(ctx, &limit)
	if err != nil {
		t.Fatalf("TagCloud(2) err = %v, want nil", err)
	}
	if len(limited) != limit {
		t.Errorf("len(TagCloud(2)) = %d, want %d", len(limited), limit)
	}
	if len(all) <= len(limited) {
		t.Errorf("len(TagCloud(nil)) = %d, want more than %d", len(all), len(limited))
	}
}

func TestTagCloudSkipsUnderscoreNames(t *testing.T) {
	d := newTestDB(t)

	tags, err := d.TagCloud(context.Background(), nil)
	if err != nil {
		t.Fatalf("TagCloud(nil) err = %v, want nil", err)
	}
	for _, tag := range tags {
		if len(tag.Name) > 0 && tag.Name[0] == '_' {
			t.Errorf("TagCloud() has %q, want no name starting with _", tag.Name)
		}
	}
}

func TestTagCloudOrdersByArticleCount(t *testing.T) {
	d := newTestDB(t)

	tags, err := d.TagCloud(context.Background(), nil)
	if err != nil {
		t.Fatalf("TagCloud(nil) err = %v, want nil", err)
	}
	for i := 1; i < len(tags); i++ {
		if tags[i-1].Articles < tags[i].Articles {
			t.Errorf("TagCloud()[%d].Articles = %d, want at least %d",
				i-1, tags[i-1].Articles, tags[i].Articles)
		}
	}
}
