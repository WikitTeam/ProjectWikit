package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/WikitTeam/ProjectWikit/internal/archive"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

func importCommand(args []string) error {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), `Usage: pwikit import [directory] [options]

Imports an unpacked wikitCLI backup: a site directory, or the directory holding
several of them next to _users. Without a directory, archive/ in the state
directory is read.

Options:
`)
		flags.PrintDefaults()
	}
	database := flags.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	slug := flags.String("site", "", "slug of the site to write into; needed once a database holds more than one")
	from := flags.String("from", "", "slug of the site inside the backup; needed when it holds more than one")
	forceTags := flags.Bool("force-tags", false, "create tags this site would otherwise refuse")
	noVotes := flags.Bool("no-votes", false, "leave the ratings behind")
	noFiles := flags.Bool("no-files", false, "leave the attachments behind")
	noAccounts := flags.Bool("no-accounts", false, "import even when the backup holds no accounts, leaving every author off")
	dataDir := flags.String("data-dir", "", "state directory holding archive/ and receiving the attachments; defaults to the directory holding the executable")
	loose, err := parseMixed(flags, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(loose) > 1 {
		return fmt.Errorf("import takes one directory, got %d", len(loose))
	}

	p, err := paths.New(*dataDir)
	if err != nil {
		return err
	}
	dir := p.Archive()
	if len(loose) == 1 {
		dir = loose[0]
	} else if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s does not exist; put the unpacked backup there or name its directory", dir)
	}
	found, err := archive.Open(dir)
	if err != nil {
		return err
	}

	ctx := context.Background()
	dsn, release, err := resolveDatabase(ctx, *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close()

	current, err := resolveSite(ctx, conn, *slug)
	if err != nil {
		return err
	}

	files := ""
	if !*noFiles {
		if err := p.EnsureBase(); err != nil {
			return err
		}
		files = p.Files()
	}
	return importArchive(ctx, conn, current, found, *from, archive.Options{
		ForceTags:       *forceTags,
		Votes:           !*noVotes,
		Files:           files,
		WithoutAccounts: *noAccounts,
	})
}

func importArchive(ctx context.Context, conn *db.DB, current *db.Site, found *archive.Archive, from string, opts archive.Options) error {
	slugs := found.Sites()
	switch {
	case from == "" && len(slugs) > 1:
		return fmt.Errorf("the backup holds %d sites, name one with -from", len(slugs))
	case from == "":
		from = slugs[0]
	case !slices.Contains(slugs, from):
		return fmt.Errorf("the backup has no site %q", from)
	}

	fmt.Printf("importing %s into %s\n", from, current.Slug)
	opts.Report = func(line string) { fmt.Println(line) }
	result, err := archive.ImportPages(ctx, conn, current.ID, found, from, opts)
	if errors.Is(err, archive.ErrNoAccounts) {
		return fmt.Errorf("%w, so nothing was imported. The accounts are in a _users directory, "+
			"usually beside the site directory; import the directory holding both, "+
			"or pass -no-accounts to import without authors", err)
	}
	fmt.Printf("%d pages, %d already there, %d revisions, %d parents, %d files, %d accounts\n",
		result.Pages, result.Skipped, result.Revisions, result.Parents, result.Files, result.Users)
	fmt.Printf("%d forum categories, %d threads, %d posts\n",
		result.Categories, result.Threads, result.Posts)
	return err
}
