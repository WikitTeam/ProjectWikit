package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

func reindex(args []string) error {
	fs := flag.NewFlagSet("reindex", flag.ContinueOnError)
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	slug := fs.String("site", "", "slug of the site to go through; needed once a database holds more than one")
	all := fs.Bool("all", false, "go through every site in this database")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	sidecar := fs.String("sidecar", os.Getenv(envSidecar), "path to the ftml sidecar binary; without it the linked-in ftml is used")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
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

	sites, err := reindexSites(ctx, conn, *slug, *all)
	if err != nil {
		return err
	}
	p, err := paths.New(*dataDir)
	if err != nil {
		return err
	}
	bundle, err := i18n.Load(p.Locales())
	if err != nil {
		return err
	}
	engine, closeEngine, err := newRenderer(*sidecar)
	if err != nil {
		return err
	}
	defer closeEngine()

	loc := bundle.Localizer(i18n.DefaultLanguage)
	store := cliRepository{data: repo.New(ctx, conn, printuser.New(loc, roles.FileIcons(p.Files())), repo.Options{Loc: loc})}
	for _, current := range sites {
		written, err := reindexSite(ctx, conn, engine, store, loc, current)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %d pages indexed\n", current.Slug, written)
	}
	return nil
}

func reindexSites(ctx context.Context, conn *db.DB, slug string, all bool) ([]*db.Site, error) {
	if !all {
		current, err := resolveSite(ctx, conn, slug)
		if err != nil {
			return nil, err
		}
		return []*db.Site{current}, nil
	}
	if slug != "" {
		return nil, errors.New("give either -site or -all, not both")
	}
	slugs, err := conn.SiteSlugs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*db.Site, 0, len(slugs))
	for _, one := range slugs {
		current, err := conn.SiteBySlug(ctx, one)
		if err != nil {
			return nil, err
		}
		out = append(out, current)
	}
	return out, nil
}

func reindexSite(ctx context.Context, conn *db.DB, engine renderer.Renderer, store cliRepository,
	loc *i18n.Localizer, current *db.Site) (int, error) {

	listed, err := conn.ListArticles(ctx, db.ListFilter{SiteID: current.ID}, 0, nil)
	if err != nil {
		return 0, err
	}
	written := 0
	for i := range listed {
		article := &listed[i]
		source, err := conn.LatestSource(ctx, article.ID)
		if errors.Is(err, db.ErrNotFound) {
			continue
		}
		if err != nil {
			return written, err
		}
		if source == "" {
			continue
		}
		indexed := article.Title + "\n\n" + source
		plaintext := indexed
		// The same fallback the editor uses, so a page the engine cannot read is
		// still searchable by its own words.
		if text, err := reindexText(ctx, engine, store, loc, conn, current, article, source); err == nil {
			plaintext = article.Title + "\n\n" + text
		}
		if err := conn.UpdateSearchIndex(ctx, article.ID, indexed, plaintext); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

func reindexText(ctx context.Context, engine renderer.Renderer, store cliRepository, loc *i18n.Localizer,
	conn *db.DB, current *db.Site, article *db.Article, source string) (string, error) {

	vars := page.NewVars(article, nil, repo.NewVarSource(ctx, conn, current), loc)
	cb := callbacks.New(loc, store)
	cb.SetSite(current.Slug)
	cb.SetPageVars(vars)
	cb.SetContext(page.NewContext(article, article, nil, nil))

	info := renderer.PageInfo{
		Page: article.Name, Category: article.Category,
		Site: current.Slug, Domain: current.Domain, Title: article.Title,
	}
	result, err := engine.RenderText(ctx, page.PreRender(source, vars), info, cb, renderer.ModeSystem)
	if err != nil {
		return "", err
	}
	return result.Body, nil
}
