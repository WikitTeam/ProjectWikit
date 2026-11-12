package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMoveFavouriteBetweenFolders(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	user := scratchUser(t, d, "probe-folder-a")
	if err := d.AddFavourite(ctx, article, user, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	folder, err := d.CreateFavouriteFolder(ctx, user, "reading", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateFavouriteFolder() err = %v, want nil", err)
	}
	if err := d.MoveFavourite(ctx, user, article, folder); err != nil {
		t.Fatalf("MoveFavourite(folder) err = %v, want nil", err)
	}
	if n, _ := d.FavouriteCountOf(ctx, user, DefaultFolder); n != 0 {
		t.Errorf("FavouriteCountOf(default) = %d, want 0", n)
	}
	if n, _ := d.FavouriteCountOf(ctx, user, folder); n != 1 {
		t.Errorf("FavouriteCountOf(folder) = %d, want 1", n)
	}
	folders, err := d.FavouriteFolders(ctx, user)
	if err != nil {
		t.Fatalf("FavouriteFolders() err = %v, want nil", err)
	}
	if len(folders) != 1 || folders[0].Count != 1 {
		t.Errorf("FavouriteFolders() = %+v, want one folder holding 1", folders)
	}
	if err := d.MoveFavourite(ctx, user, article, DefaultFolder); err != nil {
		t.Fatalf("MoveFavourite(default) err = %v, want nil", err)
	}
	if n, _ := d.FavouriteCountOf(ctx, user, DefaultFolder); n != 1 {
		t.Errorf("FavouriteCountOf(default) = %d, want 1", n)
	}
}

func TestMoveFavouriteRefusesAnotherUsersFolder(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	mine := scratchUser(t, d, "probe-folder-b")
	other := scratchUser(t, d, "probe-folder-c")
	if err := d.AddFavourite(ctx, article, mine, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	theirs, err := d.CreateFavouriteFolder(ctx, other, "theirs", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateFavouriteFolder() err = %v, want nil", err)
	}
	if err := d.MoveFavourite(ctx, mine, article, theirs); !errors.Is(err, ErrNotFound) {
		t.Errorf("MoveFavourite(other's folder) err = %v, want ErrNotFound", err)
	}
	if err := d.SetFavouriteFolderPublic(ctx, mine, theirs, false); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetFavouriteFolderPublic(other's folder) err = %v, want ErrNotFound", err)
	}
	if err := d.DeleteFavouriteFolder(ctx, mine, theirs); !errors.Is(err, ErrNotFound) {
		t.Errorf("DeleteFavouriteFolder(other's folder) err = %v, want ErrNotFound", err)
	}
}

func TestDeleteFavouriteFolderKeepsThePages(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	user := scratchUser(t, d, "probe-folder-d")
	if err := d.AddFavourite(ctx, article, user, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	folder, err := d.CreateFavouriteFolder(ctx, user, "gone", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateFavouriteFolder() err = %v, want nil", err)
	}
	if err := d.MoveFavourite(ctx, user, article, folder); err != nil {
		t.Fatalf("MoveFavourite() err = %v, want nil", err)
	}
	if err := d.DeleteFavouriteFolder(ctx, user, folder); err != nil {
		t.Fatalf("DeleteFavouriteFolder() err = %v, want nil", err)
	}
	if n, _ := d.FavouriteCountOf(ctx, user, DefaultFolder); n != 1 {
		t.Errorf("FavouriteCountOf(default) = %d, want 1", n)
	}
}

func TestPublicFavouritesFollowTheFolder(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	open := scratchArticle(t, d)
	shut := scratchArticle(t, d)
	loose := scratchArticle(t, d)
	user := scratchUser(t, d, "probe-folder-e")
	for _, a := range []int64{open, shut, loose} {
		if err := d.AddFavourite(ctx, a, user, time.Now().UTC()); err != nil {
			t.Fatalf("AddFavourite() err = %v, want nil", err)
		}
	}
	public, _ := d.CreateFavouriteFolder(ctx, user, "open", true, time.Now().UTC())
	private, _ := d.CreateFavouriteFolder(ctx, user, "shut", false, time.Now().UTC())
	if err := d.MoveFavourite(ctx, user, open, public); err != nil {
		t.Fatalf("MoveFavourite() err = %v, want nil", err)
	}
	if err := d.MoveFavourite(ctx, user, shut, private); err != nil {
		t.Fatalf("MoveFavourite() err = %v, want nil", err)
	}

	got, err := d.PublicFavourites(ctx, user, false, 100)
	if err != nil {
		t.Fatalf("PublicFavourites() err = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Article.ID != open || got[0].FolderID != public {
		t.Errorf("PublicFavourites(default private) = %+v, want only %d in folder %d", got, open, public)
	}
	got, err = d.PublicFavourites(ctx, user, true, 100)
	if err != nil {
		t.Fatalf("PublicFavourites() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Errorf("len(PublicFavourites(default public)) = %d, want 2", len(got))
	}
}

func TestRemoveFavouriteForgetsItsFolder(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	user := scratchUser(t, d, "probe-folder-f")
	if err := d.AddFavourite(ctx, article, user, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	folder, _ := d.CreateFavouriteFolder(ctx, user, "kept", false, time.Now().UTC())
	if err := d.MoveFavourite(ctx, user, article, folder); err != nil {
		t.Fatalf("MoveFavourite() err = %v, want nil", err)
	}
	if err := d.RemoveFavourite(ctx, article, user); err != nil {
		t.Fatalf("RemoveFavourite() err = %v, want nil", err)
	}
	if err := d.AddFavourite(ctx, article, user, time.Now().UTC()); err != nil {
		t.Fatalf("AddFavourite() err = %v, want nil", err)
	}
	if n, _ := d.FavouriteCountOf(ctx, user, DefaultFolder); n != 1 {
		t.Errorf("FavouriteCountOf(default) after re-adding = %d, want 1", n)
	}
}

func TestCreateFavouriteFolderRefusesADuplicateName(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	user := scratchUser(t, d, "probe-folder-g")
	if _, err := d.CreateFavouriteFolder(ctx, user, "twice", false, time.Now().UTC()); err != nil {
		t.Fatalf("CreateFavouriteFolder() err = %v, want nil", err)
	}
	if _, err := d.CreateFavouriteFolder(ctx, user, "twice", true, time.Now().UTC()); !errors.Is(err, ErrFolderExists) {
		t.Errorf("CreateFavouriteFolder(same name) err = %v, want ErrFolderExists", err)
	}
}
