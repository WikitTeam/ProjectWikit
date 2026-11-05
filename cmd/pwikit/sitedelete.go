package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

var errDeleteCancelled = errors.New("nothing was deleted")

type asker struct {
	in  *bufio.Reader
	out io.Writer
}

func (a asker) line(prompt string) string {
	fmt.Fprint(a.out, prompt)
	text, _ := a.in.ReadString('\n')
	return strings.TrimSpace(text)
}

func (a asker) yes(prompt string, fallback bool) bool {
	switch strings.ToLower(a.line(prompt)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return fallback
}

// Every step asks again, and a missing backup asks once more, because nothing
// can bring the site back afterwards.
func deleteSite(ctx context.Context, conn *db.DB, dsn, dataDir, slug string) error {
	if slug == "" {
		return errors.New("delete needs -slug")
	}
	target, err := conn.SiteBySlug(ctx, slug)
	if errors.Is(err, db.ErrNotFound) {
		return fmt.Errorf("no site with slug %q", slug)
	}
	if err != nil {
		return err
	}
	size, err := conn.SiteSize(ctx, target.ID)
	if err != nil {
		return err
	}
	slugs, err := conn.SiteSlugs(ctx)
	if err != nil {
		return err
	}
	p, err := paths.New(dataDir)
	if err != nil {
		return err
	}

	ask := asker{in: bufio.NewReader(os.Stdin), out: os.Stdout}
	fmt.Printf("Site     %s (%s)\n", target.Slug, target.Title)
	fmt.Printf("Domain   %s\n", target.Domain)
	fmt.Printf("Holds    %d pages, %d forum posts, %d files, %d members\n", size.Pages, size.Posts, size.Files, size.Members)
	if len(slugs) == 1 {
		fmt.Println("This is the only site in the database. Nothing will be left to serve.")
	}
	fmt.Println()

	backedUp := false
	if ask.yes("A backup is the only way to get this site back. Back up the whole instance now? [Y/n] ", true) {
		if err := createBackup(dsn, p.Files(), "", p.Backups(), "", false); err != nil {
			return fmt.Errorf("backup failed, %w: %w", errDeleteCancelled, err)
		}
		backedUp = true
	} else {
		backedUp = ask.yes("Is there already a backup that holds this site? [y/N] ", false)
	}

	if ask.line(fmt.Sprintf("Type the slug of the site to delete (%s): ", target.Slug)) != target.Slug {
		return errDeleteCancelled
	}
	if !backedUp {
		fmt.Println("There is no backup. Once deleted, this site can never be brought back.")
		if ask.line("Type 'delete forever' to go on: ") != "delete forever" {
			return errDeleteCancelled
		}
	}
	if !ask.yes(fmt.Sprintf("Delete %s now? This cannot be undone. [y/N] ", target.Slug), false) {
		return errDeleteCancelled
	}

	dirs, err := conn.DeleteSite(ctx, target.Slug)
	if err != nil {
		return err
	}
	root := filepath.Join(p.Files(), "media")
	left := 0
	for _, name := range dirs {
		full, err := paths.Resolve(root, media.QuoteName(name))
		if err != nil || name == "" {
			continue
		}
		if err := os.RemoveAll(full); err != nil {
			left++
		}
	}
	fmt.Printf("deleted %s\n", target.Slug)
	if left > 0 {
		fmt.Printf("%d page directories under %s could not be removed\n", left, root)
	}
	fmt.Printf("restart pwikit serve if it is running, so it stops answering for %s\n", target.Domain)
	return nil
}
