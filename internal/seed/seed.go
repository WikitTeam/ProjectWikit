// Package seed writes the pages a new site starts with.
package seed

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

//go:embed pages
var pages embed.FS

const (
	dir    = "pages"
	suffix = ".ftml"
)

type Store interface {
	ArticleByName(ctx context.Context, ref string) (*db.Article, error)
	CreateArticle(ctx context.Context, category, name, title string, authorID *int64, at time.Time) (int64, error)
	CreateArticleVersion(ctx context.Context, w db.VersionWrite) (db.Revision, error)
}

func Names() []string {
	var out []string
	fs.WalkDir(pages, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, suffix) {
			return err
		}
		out = append(out, pageName(p))
		return nil
	})
	return out
}

func pageName(p string) string {
	rel := strings.TrimSuffix(strings.TrimPrefix(p, dir+"/"), suffix)
	return strings.ReplaceAll(rel, "/", ":")
}

func Run(ctx context.Context, store Store) ([]string, error) {
	var written []string
	err := fs.WalkDir(pages, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, suffix) {
			return err
		}
		source, err := pages.ReadFile(p)
		if err != nil {
			return err
		}
		if len(source) == 0 {
			return nil
		}
		name := pageName(p)
		added, err := write(ctx, store, name, string(source))
		if err != nil {
			return err
		}
		if added {
			written = append(written, name)
		}
		return nil
	})
	return written, err
}

func write(ctx context.Context, store Store, full, source string) (bool, error) {
	if _, err := store.ArticleByName(ctx, full); err == nil {
		return false, nil
	} else if !errors.Is(err, db.ErrNotFound) {
		return false, err
	}

	category, name := split(full)

	id, err := store.CreateArticle(ctx, category, name, "", nil, time.Now())
	if err != nil {
		return false, err
	}
	_, err = store.CreateArticleVersion(ctx, db.VersionWrite{
		ArticleID: id,
		Source:    source,
		Kind:      db.LogNew,
		Comment:   "",
		At:        time.Now(),
	})
	if err != nil {
		return false, fmt.Errorf("write the first version of %q: %w", full, err)
	}
	return true, nil
}

func split(full string) (category, name string) {
	if before, after, found := strings.Cut(full, ":"); found {
		return before, after
	}
	return "_default", path.Base(full)
}
