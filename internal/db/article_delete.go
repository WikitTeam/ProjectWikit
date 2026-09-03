package db

import (
	"context"
	"fmt"
)

// Nothing in the schema cascades, so every row that points at the page has to
// be named here. The order is the one the foreign keys allow.
var articleChildren = []string{
	`DELETE FROM web_forumpostversion WHERE post_id IN (
		SELECT p.id FROM web_forumpost p
		JOIN web_forumthread t ON t.id = p.thread_id
		WHERE t.article_id = $1)`,
	`UPDATE web_forumpost SET reply_to_id = NULL WHERE thread_id IN (
		SELECT id FROM web_forumthread WHERE article_id = $1)`,
	`DELETE FROM web_forumpost WHERE thread_id IN (
		SELECT id FROM web_forumthread WHERE article_id = $1)`,
	`DELETE FROM web_usernotificationsubscription WHERE forum_thread_id IN (
		SELECT id FROM web_forumthread WHERE article_id = $1)`,
	`DELETE FROM web_forumthread WHERE article_id = $1`,
	`DELETE FROM web_usernotificationsubscription WHERE article_id = $1`,
	`DELETE FROM web_vote WHERE article_id = $1`,
	`DELETE FROM web_file WHERE article_id = $1`,
	`DELETE FROM web_article_tags WHERE article_id = $1`,
	`DELETE FROM web_article_authors WHERE article_id = $1`,
	`DELETE FROM web_articlelogentry WHERE article_id = $1`,
	`DELETE FROM web_articleversion WHERE article_id = $1`,
	`UPDATE web_articlesearchindex SET article_id = NULL WHERE article_id = $1`,
	`UPDATE web_article SET parent_id = NULL WHERE parent_id = $1`,
	`DELETE FROM web_article WHERE id = $1`,
}

var qDeleteArticle = registerAll("DeleteArticle", articleChildren)

// The files on disk are left to the caller, which is the only layer that knows
// where the state directory is.
func (d *DB) DeleteArticle(ctx context.Context, articleID int64, fullName string) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete of %d: %w", articleID, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qDropLinksFrom, fullName); err != nil {
		return fmt.Errorf("drop links of %q: %w", fullName, err)
	}
	for _, sql := range qDeleteArticle {
		if _, err := tx.Exec(ctx, sql, articleID); err != nil {
			return fmt.Errorf("delete article %d: %w", articleID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete of %d: %w", articleID, err)
	}
	return nil
}
