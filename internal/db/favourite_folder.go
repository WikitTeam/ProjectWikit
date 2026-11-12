package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Every favourite lands here until its owner moves it.
const DefaultFolder int64 = 0

const MaxFavouriteFolders = 50

var (
	ErrTooManyFolders = errors.New("too many favourite folders")
	ErrFolderExists   = errors.New("a favourite folder by that name exists")
)

type FavouriteFolder struct {
	ID     int64
	Name   string
	Public bool
	Count  int
}

var qFavouriteFolders = register("FavouriteFolders", `
SELECT fo.id, fo.name, fo.public,
       (SELECT count(*) FROM pwikit_favourite_place p
        JOIN web_articlefavourite f ON f.user_id = p.user_id AND f.article_id = p.article_id
        WHERE p.folder_id = fo.id)
FROM pwikit_favourite_folder fo
WHERE fo.user_id = $1
ORDER BY fo.created_at, fo.id`)

func (d *DB) FavouriteFolders(ctx context.Context, userID int64) ([]FavouriteFolder, error) {
	rows, err := d.pool.Query(ctx, qFavouriteFolders, userID)
	if err != nil {
		return nil, fmt.Errorf("list favourite folders of user %d: %w", userID, err)
	}
	defer rows.Close()
	var out []FavouriteFolder
	for rows.Next() {
		var f FavouriteFolder
		if err := rows.Scan(&f.ID, &f.Name, &f.Public, &f.Count); err != nil {
			return nil, fmt.Errorf("scan favourite folder: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list favourite folders of user %d: %w", userID, err)
	}
	return out, nil
}

var qFavouriteFolderNamed = register("FavouriteFolderNamed", `
SELECT EXISTS (SELECT 1 FROM pwikit_favourite_folder WHERE user_id = $1 AND name = $2)`)

var qCreateFavouriteFolder = register("CreateFavouriteFolder", `
INSERT INTO pwikit_favourite_folder (user_id, name, public, created_at)
SELECT $1, $2, $3, $4
WHERE (SELECT count(*) FROM pwikit_favourite_folder WHERE user_id = $1) < $5
RETURNING id`)

func (d *DB) CreateFavouriteFolder(ctx context.Context, userID int64, name string, public bool, at time.Time) (int64, error) {
	var taken bool
	if err := d.pool.QueryRow(ctx, qFavouriteFolderNamed, userID, name).Scan(&taken); err != nil {
		return 0, fmt.Errorf("look up favourite folder %q of user %d: %w", name, userID, err)
	}
	if taken {
		return 0, ErrFolderExists
	}
	var id int64
	err := d.pool.QueryRow(ctx, qCreateFavouriteFolder, userID, name, public, at, MaxFavouriteFolders).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrTooManyFolders
	}
	if err != nil {
		return 0, fmt.Errorf("create favourite folder of user %d: %w", userID, err)
	}
	return id, nil
}

var qSetFavouriteFolderPublic = register("SetFavouriteFolderPublic", `
UPDATE pwikit_favourite_folder SET public = $3 WHERE id = $2 AND user_id = $1`)

func (d *DB) SetFavouriteFolderPublic(ctx context.Context, userID, folderID int64, public bool) error {
	tag, err := d.pool.Exec(ctx, qSetFavouriteFolderPublic, userID, folderID, public)
	if err != nil {
		return fmt.Errorf("set favourite folder %d public: %w", folderID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var qDeleteFavouriteFolder = register("DeleteFavouriteFolder", `
DELETE FROM pwikit_favourite_folder WHERE id = $2 AND user_id = $1`)

// The pages in it go back to the default folder rather than being unfavourited.
func (d *DB) DeleteFavouriteFolder(ctx context.Context, userID, folderID int64) error {
	tag, err := d.pool.Exec(ctx, qDeleteFavouriteFolder, userID, folderID)
	if err != nil {
		return fmt.Errorf("delete favourite folder %d: %w", folderID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var (
	qPlaceFavourite = register("PlaceFavourite", `
INSERT INTO pwikit_favourite_place (user_id, article_id, folder_id)
SELECT $1, $2, $3
WHERE EXISTS (SELECT 1 FROM web_articlefavourite WHERE user_id = $1 AND article_id = $2)
  AND EXISTS (SELECT 1 FROM pwikit_favourite_folder WHERE user_id = $1 AND id = $3)
ON CONFLICT (user_id, article_id) DO UPDATE SET folder_id = EXCLUDED.folder_id`)
	qUnplaceFavourite = register("UnplaceFavourite", `
DELETE FROM pwikit_favourite_place WHERE user_id = $1 AND article_id = $2`)
)

func (d *DB) MoveFavourite(ctx context.Context, userID, articleID, folderID int64) error {
	if folderID == DefaultFolder {
		if _, err := d.pool.Exec(ctx, qUnplaceFavourite, userID, articleID); err != nil {
			return fmt.Errorf("move favourite %d to the default folder: %w", articleID, err)
		}
		return nil
	}
	tag, err := d.pool.Exec(ctx, qPlaceFavourite, userID, articleID, folderID)
	if err != nil {
		return fmt.Errorf("move favourite %d to folder %d: %w", articleID, folderID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type PublicFavourite struct {
	FolderID int64
	Favourite
}

var qPublicFavourites = register("PublicFavourites", `
SELECT COALESCE(p.folder_id, 0), a.site_id, `+prefixedArticleColumns+`, f.created_at
FROM web_articlefavourite f
JOIN web_article a ON a.id = f.article_id
LEFT JOIN pwikit_favourite_place p ON p.user_id = f.user_id AND p.article_id = f.article_id
LEFT JOIN pwikit_favourite_folder fo ON fo.id = p.folder_id
WHERE f.user_id = $1 AND (CASE WHEN p.folder_id IS NULL THEN $2 ELSE fo.public END)
ORDER BY f.created_at DESC, f.id DESC
LIMIT $3`)

// The caller still filters out pages the viewer may not see.
func (d *DB) PublicFavourites(ctx context.Context, userID int64, defaultPublic bool, limit int) ([]PublicFavourite, error) {
	rows, err := d.pool.Query(ctx, qPublicFavourites, userID, defaultPublic, limit)
	if err != nil {
		return nil, fmt.Errorf("list public favourites of user %d: %w", userID, err)
	}
	defer rows.Close()
	var out []PublicFavourite
	for rows.Next() {
		var one PublicFavourite
		a := &one.Article
		if err := rows.Scan(&one.FolderID, &one.SiteID, &a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID,
			&a.Locked, &a.CreatedAt, &a.UpdatedAt, &a.MediaName, &one.AddedAt); err != nil {
			return nil, fmt.Errorf("scan public favourite: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list public favourites of user %d: %w", userID, err)
	}
	return out, nil
}

const (
	favouritePrefSection = "profile"
	favouritePrefName    = "public_favourites"
)

// The default folder has no row of its own, so whether it is public is kept
// with the owner's other profile switches.
func (d *DB) DefaultFolderPublic(ctx context.Context, userID int64) (bool, error) {
	raw, err := d.UserPreference(ctx, userID, favouritePrefSection, favouritePrefName)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return raw == "True", err
}

func (d *DB) SetDefaultFolderPublic(ctx context.Context, userID int64, public bool) error {
	raw := "False"
	if public {
		raw = "True"
	}
	return d.SetUserPreference(ctx, userID, favouritePrefSection, favouritePrefName, raw)
}
