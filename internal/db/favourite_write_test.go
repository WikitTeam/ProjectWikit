package db

import (
	"context"
	"testing"
	"time"
)

func TestAddFavouriteCountsOnce(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	user := scratchUser(t, d, "probe-fav-a")

	for i := 0; i < 2; i++ {
		if err := d.AddFavourite(ctx, article, user, time.Now().UTC()); err != nil {
			t.Fatalf("AddFavourite() err = %v, want nil", err)
		}
	}
	got, err := d.ArticleFavouriteCount(ctx, article)
	if err != nil {
		t.Fatalf("ArticleFavouriteCount() err = %v, want nil", err)
	}
	if got != 1 {
		t.Errorf("ArticleFavouriteCount() = %d, want 1", got)
	}
}

func TestRemoveFavouriteTakesItBack(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	user := scratchUser(t, d, "probe-fav-b")

	if err := d.AddFavourite(ctx, article, user, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	if err := d.RemoveFavourite(ctx, article, user); err != nil {
		t.Fatalf("RemoveFavourite() err = %v, want nil", err)
	}
	got, err := d.HasFavourited(ctx, article, user)
	if err != nil {
		t.Fatalf("HasFavourited() err = %v, want nil", err)
	}
	if got {
		t.Errorf("HasFavourited() = true, want false")
	}
}

func TestHasFavouritedIsPerUser(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	mine := scratchUser(t, d, "probe-fav-c")
	other := scratchUser(t, d, "probe-fav-d")

	if err := d.AddFavourite(ctx, article, mine, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	yes, err := d.HasFavourited(ctx, article, mine)
	if err != nil {
		t.Fatalf("HasFavourited() err = %v, want nil", err)
	}
	if !yes {
		t.Errorf("HasFavourited(mine) = false, want true")
	}
	no, err := d.HasFavourited(ctx, article, other)
	if err != nil {
		t.Fatalf("HasFavourited() err = %v, want nil", err)
	}
	if no {
		t.Errorf("HasFavourited(other) = true, want false")
	}
}

func TestFavouritesOfComeBackNewestFirstInAWindow(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	user := scratchUser(t, d, "probe-fav-e")
	at := time.Now().UTC()

	ids := make([]int64, 0, 3)
	for i := 0; i < 3; i++ {
		article := scratchArticle(t, d)
		ids = append(ids, article)
		if err := d.AddFavourite(ctx, article, user, at.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("AddFavourite() err = %v, want nil", err)
		}
	}

	total, err := d.FavouriteCountOf(ctx, user)
	if err != nil {
		t.Fatalf("FavouriteCountOf() err = %v, want nil", err)
	}
	if total != 3 {
		t.Errorf("FavouriteCountOf() = %d, want 3", total)
	}

	first, err := d.FavouritesOf(ctx, user, 0, 2)
	if err != nil {
		t.Fatalf("FavouritesOf() err = %v, want nil", err)
	}
	if len(first) != 2 {
		t.Fatalf("len(FavouritesOf(0, 2)) = %d, want 2", len(first))
	}
	if first[0].Article.ID != ids[2] {
		t.Errorf("FavouritesOf()[0].ID = %d, want %d", first[0].Article.ID, ids[2])
	}

	second, err := d.FavouritesOf(ctx, user, 2, 2)
	if err != nil {
		t.Fatalf("FavouritesOf() err = %v, want nil", err)
	}
	if len(second) != 1 {
		t.Fatalf("len(FavouritesOf(2, 2)) = %d, want 1", len(second))
	}
	if second[0].Article.ID != ids[0] {
		t.Errorf("FavouritesOf(2, 2)[0].ID = %d, want %d", second[0].Article.ID, ids[0])
	}
}
