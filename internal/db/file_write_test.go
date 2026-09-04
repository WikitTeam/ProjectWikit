package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func scratchFile(t *testing.T, d *DB, articleID int64, name string, size int64) int64 {
	t.Helper()
	id, err := d.AddArticleFile(context.Background(), FileWrite{
		ArticleID: articleID,
		Name:      name,
		MediaName: name + ".bin",
		MimeType:  "application/octet-stream",
		Size:      size,
		At:        time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("AddArticleFile(%q) err = %v, want nil", name, err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_file WHERE id = $1`, id); err != nil {
			t.Errorf("clean up scratch file err = %v, want nil", err)
		}
	})
	return id
}

func TestAddArticleFileIsFoundByName(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	id := scratchFile(t, d, article, "probe-upload.txt", 12)

	found, err := d.LiveFileNamed(ctx, article, "probe-upload.txt")
	if err != nil {
		t.Fatalf("LiveFileNamed() err = %v, want nil", err)
	}
	if found != id {
		t.Errorf("LiveFileNamed() = %d, want %d", found, id)
	}
}

func TestLiveFileNamedMissesAnotherName(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	scratchFile(t, d, article, "probe-upload.txt", 12)

	if _, err := d.LiveFileNamed(ctx, article, "probe-other.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("LiveFileNamed() err = %v, want ErrNotFound", err)
	}
}

func TestLiveFileNamedMissesADeletedFile(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	id := scratchFile(t, d, article, "probe-upload.txt", 12)

	if _, _, err := d.SoftDeleteFile(ctx, id, time.Now().UTC(), nil); err != nil {
		t.Fatalf("SoftDeleteFile() err = %v, want nil", err)
	}
	if _, err := d.LiveFileNamed(ctx, article, "probe-upload.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("LiveFileNamed() err = %v, want ErrNotFound", err)
	}
}

func TestArticleFileListLeavesOutADeletedFile(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	kept := scratchFile(t, d, article, "probe-kept.txt", 4)
	gone := scratchFile(t, d, article, "probe-gone.txt", 4)

	if _, _, err := d.SoftDeleteFile(ctx, gone, time.Now().UTC(), nil); err != nil {
		t.Fatalf("SoftDeleteFile() err = %v, want nil", err)
	}
	list, err := d.ArticleFileList(ctx, article)
	if err != nil {
		t.Fatalf("ArticleFileList() err = %v, want nil", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(ArticleFileList()) = %d, want 1", len(list))
	}
	if list[0].ID != kept {
		t.Errorf("ArticleFileList()[0].ID = %d, want %d", list[0].ID, kept)
	}
}

func TestArticleFileListOrdersByUpload(t *testing.T) {
	d := writeTestDB(t)
	article := scratchArticle(t, d)
	first := scratchFile(t, d, article, "zzz-first.txt", 4)
	second := scratchFile(t, d, article, "aaa-second.txt", 4)

	list, err := d.ArticleFileList(context.Background(), article)
	if err != nil {
		t.Fatalf("ArticleFileList() err = %v, want nil", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(ArticleFileList()) = %d, want 2", len(list))
	}
	if list[0].ID != first {
		t.Errorf("ArticleFileList()[0].ID = %d, want %d", list[0].ID, first)
	}
	if list[1].ID != second {
		t.Errorf("ArticleFileList()[1].ID = %d, want %d", list[1].ID, second)
	}
}

func TestArticleFileListCarriesWhatWasWritten(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	at := time.Date(2026, 3, 4, 5, 6, 7, 891000000, time.UTC)

	id, err := d.AddArticleFile(ctx, FileWrite{
		ArticleID: article,
		Name:      "probe-detail.pdf",
		MediaName: "8ab0e9f2.pdf",
		MimeType:  "application/pdf",
		Size:      4096,
		At:        at,
	})
	if err != nil {
		t.Fatalf("AddArticleFile() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_file WHERE id = $1`, id); err != nil {
			t.Errorf("clean up scratch file err = %v, want nil", err)
		}
	})

	list, err := d.ArticleFileList(ctx, article)
	if err != nil {
		t.Fatalf("ArticleFileList() err = %v, want nil", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(ArticleFileList()) = %d, want 1", len(list))
	}
	got := list[0]
	if got.Name != "probe-detail.pdf" {
		t.Errorf("ArticleFileList()[0].Name = %q, want %q", got.Name, "probe-detail.pdf")
	}
	if got.MimeType != "application/pdf" {
		t.Errorf("ArticleFileList()[0].MimeType = %q, want %q", got.MimeType, "application/pdf")
	}
	if got.Size != 4096 {
		t.Errorf("ArticleFileList()[0].Size = %d, want 4096", got.Size)
	}
	if !got.CreatedAt.Equal(at) {
		t.Errorf("ArticleFileList()[0].CreatedAt = %v, want %v", got.CreatedAt.UTC(), at)
	}
	if got.AuthorID != nil {
		t.Errorf("ArticleFileList()[0].AuthorID = %v, want nil", *got.AuthorID)
	}
}

func TestFileByIDReportsADeletedFile(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	id := scratchFile(t, d, article, "probe-gone.txt", 4)

	if _, _, err := d.SoftDeleteFile(ctx, id, time.Now().UTC(), nil); err != nil {
		t.Fatalf("SoftDeleteFile() err = %v, want nil", err)
	}
	found, err := d.FileByID(ctx, id)
	if err != nil {
		t.Fatalf("FileByID() err = %v, want nil", err)
	}
	if !found.Deleted {
		t.Errorf("FileByID().Deleted = false, want true")
	}
	if found.ArticleID != article {
		t.Errorf("FileByID().ArticleID = %d, want %d", found.ArticleID, article)
	}
}

func TestFileByIDOfNothing(t *testing.T) {
	d := writeTestDB(t)
	if _, err := d.FileByID(context.Background(), -1); !errors.Is(err, ErrNotFound) {
		t.Errorf("FileByID(-1) err = %v, want ErrNotFound", err)
	}
}

func TestFileSpaceUsageCountsADeletedFileOnlyInTheTotal(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)

	beforeLive, beforeTotal, err := d.FileSpaceUsage(ctx)
	if err != nil {
		t.Fatalf("FileSpaceUsage() err = %v, want nil", err)
	}
	scratchFile(t, d, article, "probe-kept.txt", 100)
	gone := scratchFile(t, d, article, "probe-gone.txt", 700)
	if _, _, err := d.SoftDeleteFile(ctx, gone, time.Now().UTC(), nil); err != nil {
		t.Fatalf("SoftDeleteFile() err = %v, want nil", err)
	}

	live, total, err := d.FileSpaceUsage(ctx)
	if err != nil {
		t.Fatalf("FileSpaceUsage() err = %v, want nil", err)
	}
	if live-beforeLive != 100 {
		t.Errorf("FileSpaceUsage() live grew by %d, want 100", live-beforeLive)
	}
	if total-beforeTotal != 800 {
		t.Errorf("FileSpaceUsage() total grew by %d, want 800", total-beforeTotal)
	}
}
