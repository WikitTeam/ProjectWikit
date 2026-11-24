package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/quota"
)

type limitChange struct {
	maxFile, siteQuota string
	clear              bool
}

func (c limitChange) empty() bool {
	return c.maxFile == "" && c.siteQuota == "" && !c.clear
}

func siteLimits(ctx context.Context, conn *db.DB, dataDir, slug string, change limitChange) error {
	p, err := paths.New(dataDir)
	if err != nil {
		return err
	}
	cfg, err := config.Load(p.Config())
	if err != nil {
		return err
	}
	if !change.empty() {
		if slug == "" {
			return errors.New("name the site with -slug")
		}
		target, err := conn.SiteBySlug(ctx, slug)
		if err != nil {
			return fmt.Errorf("site %q: %w", slug, err)
		}
		if err := setLimits(ctx, conn, target.ID, change); err != nil {
			return err
		}
	}
	slugs := []string{slug}
	if slug == "" {
		if slugs, err = conn.SiteSlugs(ctx); err != nil {
			return err
		}
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tMAX FILE\tFROM\tSITE QUOTA\tFROM\tUSED")
	for _, s := range slugs {
		target, err := conn.SiteBySlug(ctx, s)
		if err != nil {
			return fmt.Errorf("site %q: %w", s, err)
		}
		row, err := effectiveLimits(ctx, conn, cfg, target)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s, row.maxFile, row.maxFrom, row.quota, row.quotaFrom, quota.Format(row.used))
	}
	return w.Flush()
}

func setLimits(ctx context.Context, conn *db.DB, siteID int64, change limitChange) error {
	if change.clear {
		return conn.SetSiteUploadLimit(ctx, siteID, nil, nil)
	}
	maxFile, siteQuota, err := conn.SiteUploadLimit(ctx, siteID)
	if err != nil {
		return err
	}
	given, err := quota.ParseRule(change.maxFile, change.siteQuota)
	if err != nil {
		return err
	}
	if given.MaxFile != nil {
		maxFile = given.MaxFile
	}
	if given.SiteQuota != nil {
		siteQuota = given.SiteQuota
	}
	return conn.SetSiteUploadLimit(ctx, siteID, maxFile, siteQuota)
}

type limitRow struct {
	maxFile, maxFrom, quota, quotaFrom string
	used                               int64
}

// Each value is named by the layer it came from, so someone puzzled by a limit
// sees whether to look in pwikit.toml or undo a command.
func effectiveLimits(ctx context.Context, conn *db.DB, cfg config.File, target *db.Site) (limitRow, error) {
	maxFile, siteQuota, err := conn.SiteUploadLimit(ctx, target.ID)
	if err != nil {
		return limitRow{}, err
	}
	instance, err := quota.ParseRule(cfg.Uploads.MaxFile, cfg.Uploads.SiteQuota)
	if err != nil {
		return limitRow{}, fmt.Errorf("[uploads] %w", err)
	}
	own := cfg.Uploads.Sites[target.Slug]
	perSite, err := quota.ParseRule(own.MaxFile, own.SiteQuota)
	if err != nil {
		return limitRow{}, fmt.Errorf("[uploads.sites.%s] %w", target.Slug, err)
	}
	row := limitRow{maxFile: shown(quota.DefaultMaxFile), maxFrom: "default", quota: shown(0), quotaFrom: "default"}
	layers := []struct {
		name string
		rule quota.Rule
	}{
		{"[uploads]", instance},
		{"[uploads.sites." + target.Slug + "]", perSite},
		{"command line", quota.Rule{MaxFile: maxFile, SiteQuota: siteQuota}},
	}
	for _, l := range layers {
		if l.rule.MaxFile != nil {
			row.maxFile, row.maxFrom = shown(*l.rule.MaxFile), l.name
		}
		if l.rule.SiteQuota != nil {
			row.quota, row.quotaFrom = shown(*l.rule.SiteQuota), l.name
		}
	}
	if row.used, err = conn.SiteFileUsage(ctx, target.ID); err != nil {
		return limitRow{}, err
	}
	return row, nil
}

func shown(n int64) string {
	if n == 0 {
		return "none"
	}
	return quota.Format(n)
}
