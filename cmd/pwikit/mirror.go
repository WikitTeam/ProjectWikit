package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/mirror"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikidotsrc"
)

const (
	mirrorPerSite = 5
	mirrorTotal   = 20
)

func mirrorUsage() {
	fmt.Fprint(os.Stderr, `Usage: pwikit mirror <command> [options] [:wiki:page ...]

Pages included from another Wikidot wiki are kept as read-only mirrors when no
site of this instance has that name.

  scan     find every [[include :wiki:page]] in the pages of a site
  missing  list included pages that are neither on this instance nor mirrored
  list     list the mirrors and how many pages of a site include each
  refs     list the pages that include the named mirrors
  show     print the source of the named mirrors
  fetch    fetch every missing page, following the pages they include
  refetch  fetch the named mirrors again, or every mirror with -all
  delete   delete the named mirrors

Options:
  -site      site to look at; every site when left out
  -all       with refetch, every mirror
  -per-site  requests one Wikidot wiki gets at once
  -total     requests in flight at once
  -database  PostgreSQL connection string
  -data-dir  state directory; defaults to the directory holding the executable
`)
}

func mirrorCommand(args []string) error {
	commands := []string{"scan", "missing", "list", "refs", "show", "fetch", "refetch", "delete"}
	if len(args) == 0 || !slices.Contains(commands, args[0]) {
		mirrorUsage()
		return errors.New("unknown mirror subcommand")
	}
	fs := flag.NewFlagSet("mirror "+args[0], flag.ContinueOnError)
	fs.Usage = mirrorUsage
	slug := fs.String("site", "", "site to look at")
	all := fs.Bool("all", false, "every mirror")
	perSite := fs.Int("per-site", mirrorPerSite, "requests one Wikidot wiki gets at once")
	total := fs.Int("total", mirrorTotal, "requests in flight at once")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	dataDir := fs.String("data-dir", "", "state directory")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	named, err := mirrorTargets(fs.Args())
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

	sites, err := reindexSites(ctx, conn, *slug, *slug == "")
	if err != nil {
		return err
	}
	client := wikidotsrc.New(*perSite, *total)

	switch args[0] {
	case "scan":
		for _, s := range sites {
			sum, err := mirror.Scan(ctx, conn, s.ID)
			if err != nil {
				return err
			}
			fmt.Printf("%s: %d pages include %d off-site pages; %d on this instance, %d mirrored, %d missing\n",
				s.Slug, sum.Pages, sum.Refs, sum.Instance, sum.Mirrored, sum.Missing)
		}
	case "missing":
		rows, err := conn.MissingIncludes(ctx, siteFilter(*slug, sites))
		if err != nil {
			return err
		}
		for _, m := range rows {
			state := m.Status
			if state == "" {
				state = "not fetched"
			}
			fmt.Printf(":%s:%s\t%s\t%s\t%s\n", m.Wiki, m.Page, state, m.Problem, strings.Join(m.Pages, " "))
		}
	case "list":
		uses := map[int64]int{}
		var mirrors []db.Mirror
		for _, s := range sites {
			found, err := conn.Mirrors(ctx, s.ID)
			if err != nil {
				return err
			}
			for _, m := range found {
				uses[m.ID] += m.Refs
			}
			mirrors = found
		}
		for _, m := range mirrors {
			fmt.Printf(":%s:%s\t%s\t%s\t%d\t%s\n", m.Wiki, m.Page, m.Status,
				m.FetchedAt.Local().Format(time.DateTime), uses[m.ID], m.Problem)
		}
	case "refs":
		for _, t := range named {
			for _, s := range sites {
				pages, err := conn.MirrorReferrers(ctx, s.ID, t.Wiki, t.Page)
				if err != nil {
					return err
				}
				for _, p := range pages {
					fmt.Printf("%s\t%s\t%s\n", t, s.Slug, p)
				}
			}
		}
	case "show":
		for _, t := range named {
			m, err := conn.MirrorByName(ctx, t.Wiki, t.Page)
			if err != nil {
				return fmt.Errorf("%s: %w", t, err)
			}
			fmt.Println(m.Source)
		}
	case "fetch":
		targets, err := mirror.Missing(ctx, conn, siteFilter(*slug, sites))
		if err != nil {
			return err
		}
		return runMirrorFetch(ctx, conn, client, targets)
	case "refetch":
		targets := named
		if *all && len(sites) > 0 {
			mirrors, err := conn.Mirrors(ctx, sites[0].ID)
			if err != nil {
				return err
			}
			for _, m := range mirrors {
				targets = append(targets, mirror.Target{Wiki: m.Wiki, Page: m.Page})
			}
		}
		return runMirrorFetch(ctx, conn, client, targets)
	case "delete":
		for _, t := range named {
			m, err := conn.MirrorByName(ctx, t.Wiki, t.Page)
			if err != nil {
				return fmt.Errorf("%s: %w", t, err)
			}
			if err := conn.DeleteMirror(ctx, m.ID); err != nil {
				return err
			}
			fmt.Printf("deleted %s\n", t)
		}
	}
	return nil
}

func mirrorTargets(args []string) ([]mirror.Target, error) {
	out := make([]mirror.Target, 0, len(args))
	for _, arg := range args {
		wiki, page := wikidot.SplitSiteRef(arg)
		if wiki == "" {
			return nil, fmt.Errorf("%q is not :wiki:page", arg)
		}
		out = append(out, mirror.Key(wiki, page))
	}
	return out, nil
}

func siteFilter(slug string, sites []*db.Site) *int64 {
	if slug == "" {
		return nil
	}
	return &sites[0].ID
}

func runMirrorFetch(ctx context.Context, conn *db.DB, client *wikidotsrc.Client, targets []mirror.Target) error {
	if len(targets) == 0 {
		fmt.Println("nothing to fetch")
		return nil
	}
	res, err := mirror.Fetch(ctx, conn, client, targets, func(p mirror.Progress) {
		fmt.Printf("\r%d/%d fetched, %d failed  %-60s", p.Done, p.Total, p.Failed, p.Current)
	})
	fmt.Println()
	if err != nil {
		return err
	}
	fmt.Printf("%d fetched, %d failed\n", res.Fetched, res.Failed)
	return nil
}
