package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var qSiteUploadLimit = register("SiteUploadLimit", `
SELECT max_file, site_quota FROM pwikit_site_upload_limit WHERE site_id = $1`)

func (d *DB) SiteUploadLimit(ctx context.Context, siteID int64) (maxFile, siteQuota *int64, err error) {
	err = d.pool.QueryRow(ctx, qSiteUploadLimit, siteID).Scan(&maxFile, &siteQuota)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read the upload limits of site %d: %w", siteID, err)
	}
	return maxFile, siteQuota, nil
}

var (
	qSetSiteUploadLimit = register("SetSiteUploadLimit", `
INSERT INTO pwikit_site_upload_limit (site_id, max_file, site_quota) VALUES ($1, $2, $3)
ON CONFLICT (site_id) DO UPDATE SET max_file = EXCLUDED.max_file, site_quota = EXCLUDED.site_quota`)
	qClearSiteUploadLimit = register("ClearSiteUploadLimit", `
DELETE FROM pwikit_site_upload_limit WHERE site_id = $1`)
)

func (d *DB) SetSiteUploadLimit(ctx context.Context, siteID int64, maxFile, siteQuota *int64) error {
	var err error
	if maxFile == nil && siteQuota == nil {
		_, err = d.pool.Exec(ctx, qClearSiteUploadLimit, siteID)
	} else {
		_, err = d.pool.Exec(ctx, qSetSiteUploadLimit, siteID, maxFile, siteQuota)
	}
	if err != nil {
		return fmt.Errorf("set the upload limits of site %d: %w", siteID, err)
	}
	return nil
}

// Every row counts, deleted or not, because a deleted file only leaves the disk
// once pwikit removes it, and a row whose file is gone has been removed too.
var qSiteFileUsage = register("SiteFileUsage", `
SELECT COALESCE(SUM(f.size), 0)
FROM web_file f JOIN web_article a ON a.id = f.article_id
WHERE a.site_id = $1`)

func (d *DB) SiteFileUsage(ctx context.Context, siteID int64) (int64, error) {
	var used int64
	if err := d.pool.QueryRow(ctx, qSiteFileUsage, siteID).Scan(&used); err != nil {
		return 0, fmt.Errorf("sum the files of site %d: %w", siteID, err)
	}
	return used, nil
}

type StoredFile struct {
	Name             string
	ArticleMediaName string
	MediaName        string
}

var qDropFile = register("DropFile", `
DELETE FROM web_file f USING web_article a
WHERE f.id = $1 AND a.id = f.article_id AND a.site_id = $2
RETURNING f.name, a.media_name, f.media_name`)

// The row goes with the file, so reverting a page later finds nothing to bring
// back rather than a name whose bytes are gone.
func (d *DB) DropFile(ctx context.Context, siteID, fileID int64) (StoredFile, bool, error) {
	var f StoredFile
	err := d.pool.QueryRow(ctx, qDropFile, fileID, siteID).Scan(&f.Name, &f.ArticleMediaName, &f.MediaName)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredFile{}, false, nil
	}
	if err != nil {
		return StoredFile{}, false, fmt.Errorf("drop file %d: %w", fileID, err)
	}
	return f, true, nil
}

var qSiteFileStats = register("SiteFileStats", `
SELECT count(*), COALESCE(SUM(f.size), 0)
FROM web_file f JOIN web_article a ON a.id = f.article_id
WHERE a.site_id = $1 AND f.deleted_at IS NULL`)

func (d *DB) SiteFileStats(ctx context.Context, siteID int64) (count int, size int64, err error) {
	if err := d.pool.QueryRow(ctx, qSiteFileStats, siteID).Scan(&count, &size); err != nil {
		return 0, 0, fmt.Errorf("count the files of site %d: %w", siteID, err)
	}
	return count, size, nil
}
