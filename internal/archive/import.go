package archive

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

type Options struct {
	// ForceTags creates tags the site would otherwise refuse, which is what an
	// import of somebody else's wiki almost always needs.
	ForceTags bool

	// Votes is off when only the text is wanted, since ratings are the part an
	// owner most often means to start over.
	Votes bool

	// Report is called with a line worth showing while a long import runs.
	Report func(string)
}

type Result struct {
	Users     int
	Pages     int
	Skipped   int
	Revisions int
	Parents   int
}

// ImportPages writes the pages of one site in the archive, skipping any page the
// database already has. Nothing already here is edited.
func ImportPages(ctx context.Context, d *db.DB, siteID int64, a *Archive, slug string, opts Options) (Result, error) {
	var out Result

	pages, err := a.Pages(slug)
	if err != nil {
		return out, err
	}
	byWikidot, err := importUsers(ctx, d, a)
	if err != nil {
		return out, err
	}
	out.Users = len(byWikidot)
	report(opts, fmt.Sprintf("%d accounts, %d pages", out.Users, len(pages)))

	local := make(map[string]int64, len(pages))
	for _, page := range pages {
		existing, err := d.ArticleByName(ctx, siteID, page.Name)
		if err == nil {
			out.Skipped++
			local[wikidot.Normalize(page.Name)] = existing.ID
			continue
		}
		if !errors.Is(err, db.ErrNotFound) {
			return out, err
		}

		id, revisions, err := importPage(ctx, d, siteID, a, slug, page, byWikidot, opts)
		if err != nil {
			return out, err
		}
		local[wikidot.Normalize(page.Name)] = id
		out.Pages++
		out.Revisions += revisions
		if out.Pages%200 == 0 {
			report(opts, fmt.Sprintf("%d of %d pages", out.Pages, len(pages)))
		}
	}

	for _, page := range pages {
		if page.Parent == "" {
			continue
		}
		child, ok := local[wikidot.Normalize(page.Name)]
		if !ok {
			continue
		}
		parent, ok := local[wikidot.Normalize(page.Parent)]
		if !ok {
			continue
		}
		if err := d.SetImportedParent(ctx, siteID, child, parent); err != nil {
			return out, err
		}
		out.Parents++
	}
	return out, nil
}

func importUsers(ctx context.Context, d *db.DB, a *Archive) (map[int64]int64, error) {
	found, err := a.Users()
	if err != nil {
		return nil, err
	}
	list := make([]db.ImportUser, 0, len(found))
	for _, u := range found {
		list = append(list, db.ImportUser{
			WikidotID:   u.ID,
			Username:    u.Username,
			DisplayName: u.FullName,
		})
	}
	return d.EnsureWikidotUsers(ctx, list, time.Now().UTC())
}

func importPage(ctx context.Context, d *db.DB, siteID int64, a *Archive, slug string,
	page Page, byWikidot map[int64]int64, opts Options) (int64, int, error) {

	sources, err := a.Sources(slug, page)
	if err != nil {
		return 0, 0, err
	}

	category, name := wikidot.Split(wikidot.Normalize(page.Name))
	oldest := page.Revisions[len(page.Revisions)-1]
	newest := page.Revisions[0]

	write := db.ImportArticle{
		Category:  category,
		Name:      name,
		Title:     page.Title,
		Locked:    page.Locked,
		CreatedAt: time.Unix(oldest.Stamp, 0).UTC(),
		UpdatedAt: time.Unix(newest.Stamp, 0).UTC(),
		AuthorID:  localUser(byWikidot, oldest.Author),
	}

	// The archive lists the newest revision first and the history reads the
	// other way round.
	for i := len(page.Revisions) - 1; i >= 0; i-- {
		rev := page.Revisions[i]
		one := db.ImportRevision{
			Number:  rev.Number,
			UserID:  localUser(byWikidot, rev.Author),
			Comment: rev.Comment,
			At:      time.Unix(rev.Stamp, 0).UTC(),
			IsNew:   rev.IsNew(),
		}
		if source, ok := sources[rev.Number]; ok {
			one.Source = &source
		}
		write.Revisions = append(write.Revisions, one)
	}

	if opts.Votes {
		for _, vote := range page.Votings {
			user := localUser(byWikidot, vote.UserID)
			if user == nil {
				continue
			}
			write.Votes = append(write.Votes, db.ImportVote{UserID: *user, Rate: vote.Value})
		}
	}
	if len(page.Tags) > 0 {
		ids, err := d.EnsureTags(ctx, siteID, page.Tags, opts.ForceTags)
		if err != nil {
			return 0, 0, err
		}
		write.TagIDs = ids
	}

	id, err := d.ImportArticle(ctx, siteID, write)
	if err != nil {
		return 0, 0, err
	}
	return id, len(write.Revisions), nil
}

// An author the archive never described is left off rather than invented, which
// shows the revision as the system's own.
func localUser(byWikidot map[int64]int64, wikidotID int64) *int64 {
	if id, ok := byWikidot[wikidotID]; ok {
		return &id
	}
	return nil
}

func report(opts Options, line string) {
	if opts.Report != nil {
		opts.Report(line)
	}
}
