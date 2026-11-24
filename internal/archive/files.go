package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

func (im *importer) importFiles(ctx context.Context, page Page, articleID int64, articleMedia string) (int, error) {
	if im.opts.Files == nil || len(page.Files) == 0 {
		return 0, nil
	}
	written := 0
	for _, file := range page.Files {
		from := im.archive.FilePath(im.slug, page.Name, file.ID)
		if from == "" {
			report(im.opts, fmt.Sprintf("missing attachment %s/%s", page.Name, file.Name))
			continue
		}
		stored, err := storedName(file.Name)
		if err != nil {
			return written, err
		}
		to := storage.Join("media", media.QuoteName(articleMedia), media.QuoteName(stored))
		if err := copyFile(ctx, im.opts.Files, from, to); err != nil {
			return written, err
		}
		_, err = im.db.AddArticleFile(ctx, db.FileWrite{
			ArticleID: articleID,
			Name:      file.Name,
			MediaName: stored,
			MimeType:  file.Mime,
			Size:      file.Size,
			AuthorID:  localUser(im.users, file.Author),
			At:        time.Unix(file.Stamp, 0).UTC(),
		})
		if err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// The stored name keeps the extension and nothing else, because the name a
// visitor typed is answered from the database rather than from the disk.
func storedName(name string) (string, error) {
	unique, err := db.MediaName()
	if err != nil {
		return "", err
	}
	return unique + filepath.Ext(name), nil
}

func copyFile(ctx context.Context, store storage.Store, from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	return store.Put(ctx, to, in, info.Size())
}
