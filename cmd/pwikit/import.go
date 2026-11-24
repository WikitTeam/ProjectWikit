package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/archive"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
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
	noTags := flags.Bool("no-tags", false, "leave the tags behind")
	noVotes := flags.Bool("no-votes", false, "leave the ratings behind")
	noFiles := flags.Bool("no-files", false, "leave the attachments behind")
	noAccounts := flags.Bool("no-accounts", false, "create no accounts, even when the backup holds some, leaving every author off")
	ownUsers := flags.Bool("own-users", false, "read accounts only from the _users inside each site directory, not the shared one beside them")
	usedUsers := flags.Bool("used-users", false, "create accounts only for the users the imported pages, ratings, attachments and forum name")
	backfill := flags.Bool("user-backfill", false, "on pages already imported, fill in the authors and ratings that are missing")
	update := flags.Bool("update", false, "bring pages already imported up to the backup, and add the ratings, attachments and forum posts they are missing")
	yes := flags.Bool("yes", false, "with -update, go ahead without asking")
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
	if *backfill && *noAccounts {
		return errors.New("-user-backfill needs accounts, so it cannot go with -no-accounts")
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
	if *ownUsers {
		found.DropSharedUsers()
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

	bundle, err := i18n.Load(p.Locales())
	if err != nil {
		return err
	}
	loc := bundle.Localizer(current.Language)

	var files storage.Store
	if !*noFiles {
		if err := p.EnsureBase(); err != nil {
			return err
		}
		set, err := instanceStorage(p)
		if err != nil {
			return err
		}
		files = set.For(current.Slug).Store
	}
	return importArchive(ctx, conn, current, found, *from, importSteps{backfill: *backfill, update: *update, yes: *yes}, archive.Options{
		Tags:            !*noTags,
		Votes:           !*noVotes,
		Files:           files,
		WithoutAccounts: *noAccounts,
		UsedUsersOnly:   *usedUsers,
		DeletedName: func(id int64) string {
			return loc.T("user-deleted-wikidot", "id", id)
		},
	})
}

type importSteps struct {
	backfill bool
	update   bool
	yes      bool
}

func importArchive(ctx context.Context, conn *db.DB, current *db.Site, found *archive.Archive, from string, steps importSteps, opts archive.Options) error {
	slugs := found.Sites()
	switch {
	case from == "" && len(slugs) > 1:
		return fmt.Errorf("the backup holds %d sites, name one with -from", len(slugs))
	case from == "":
		from = slugs[0]
	case !slices.Contains(slugs, from):
		return fmt.Errorf("the backup has no site %q", from)
	}

	if steps.update {
		if err := confirmUpdate(ctx, conn, current, found, from, steps.yes); err != nil {
			return err
		}
	}

	fmt.Printf("importing %s into %s\n", from, current.Slug)
	opts.Report = func(line string) { fmt.Println(line) }
	result, err := archive.ImportPages(ctx, conn, current.ID, found, from, opts)
	if errors.Is(err, archive.ErrNoAccounts) {
		return fmt.Errorf("%w, so nothing was imported. The accounts are in a _users directory, "+
			"which belongs beside the site directory; put it there, "+
			"or pass -no-accounts to import without authors", err)
	}
	fmt.Printf("%d pages, %d already there, %d revisions, %d parents, %d files, %d accounts\n",
		result.Pages, result.Skipped, result.Revisions, result.Parents, result.Files, result.Users)
	fmt.Printf("%d forum categories, %d threads, %d posts\n",
		result.Categories, result.Threads, result.Posts)
	printRenumbered(result.Renumbered)
	if err != nil {
		return err
	}
	fmt.Printf("run `pwikit reindex -site %s` to make the imported pages searchable and record their links\n", current.Slug)
	if steps.update {
		done, err := archive.ApplyUpdate(ctx, conn, current.ID, found, from, opts)
		fmt.Printf("updated %d pages with %d revisions, added %d ratings, %d attachments, "+
			"%d forum categories, %d threads and %d posts\n",
			done.Pages, done.Revisions, done.Votes, done.Files, done.Categories, done.Threads, done.Posts)
		printEdited(done.Edited)
		printRenumbered(done.Renumbered)
		if err != nil {
			return err
		}
	}
	if !steps.backfill {
		return nil
	}
	fixed, err := archive.BackfillUsers(ctx, conn, current.ID, found, from, opts)
	fmt.Printf("filled in %d revision authors, %d page authors, %d ratings, %d attachment authors, "+
		"%d thread authors, %d post authors, %d post version authors\n",
		fixed.Revisions, fixed.Authors, fixed.Votes, fixed.Files, fixed.Threads, fixed.Posts, fixed.Versions)
	return err
}

func confirmUpdate(ctx context.Context, conn *db.DB, current *db.Site, found *archive.Archive, from string, yes bool) error {
	plan, err := archive.PlanUpdate(ctx, conn, current.ID, found, from)
	if err != nil {
		return err
	}
	fmt.Printf("%s holds %d new pages, %d pages with newer revisions and %d pages already up to date\n",
		from, plan.New, len(plan.Update), plan.Current)
	printEdited(plan.Edited)
	fmt.Println("Missing ratings, attachments, forum threads and posts will be added to every page and thread already here.")
	if yes {
		return nil
	}
	fmt.Print("Continue? [y/N] ")
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && answer == "" {
		return errors.New("cancelled")
	}
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		return errors.New("cancelled")
	}
	return nil
}

var renumberedPrefix = map[string]string{
	db.RenumberedCategory: "c-",
	db.RenumberedThread:   "t-",
	db.RenumberedPost:     "post-",
}

func printRenumbered(moved []db.Renumbered) {
	if len(moved) == 0 {
		return
	}
	fmt.Printf("%d forum numbers from Wikidot were already taken here and got new ones:\n", len(moved))
	for _, one := range moved {
		prefix := renumberedPrefix[one.Kind]
		fmt.Printf("  %s%d -> %s%d\n", prefix, one.From, prefix, one.To)
	}
}

func printEdited(names []string) {
	if len(names) == 0 {
		return
	}
	fmt.Printf("%d pages were changed on this site after the import and are left alone:\n", len(names))
	for _, name := range names {
		fmt.Println("  " + name)
	}
}
