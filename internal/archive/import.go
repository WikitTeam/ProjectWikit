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

	// Files is the state directory attachments are copied into. Empty leaves
	// them behind.
	Files string

	// Report is called with a line worth showing while a long import runs.
	Report func(string)
}

type Result struct {
	Users     int
	Pages     int
	Skipped   int
	Revisions int
	Parents   int
	Files     int

	Categories int
	Threads    int
	Posts      int
}

type importer struct {
	db      *db.DB
	archive *Archive
	slug    string
	siteID  int64
	opts    Options
	users   map[int64]int64
}

// ImportPages writes the pages of one site in the archive, skipping any page the
// database already has. Nothing already here is edited.
func ImportPages(ctx context.Context, d *db.DB, siteID int64, a *Archive, slug string, opts Options) (Result, error) {
	var out Result
	im := &importer{db: d, archive: a, slug: slug, siteID: siteID, opts: opts}

	pages, err := a.Pages(slug)
	if err != nil {
		return out, err
	}
	im.users, err = im.importUsers(ctx)
	if err != nil {
		return out, err
	}
	out.Users = len(im.users)
	report(opts, fmt.Sprintf("%d accounts, %d pages", out.Users, len(pages)))

	local := make(map[string]int64, len(pages))
	byThread := map[int64]int64{}
	for _, page := range pages {
		existing, err := d.ArticleByName(ctx, siteID, page.Name)
		if err == nil {
			out.Skipped++
			local[wikidot.Normalize(page.Name)] = existing.ID
			byThread[page.ThreadID] = existing.ID
			continue
		}
		if !errors.Is(err, db.ErrNotFound) {
			return out, err
		}

		id, media, revisions, err := im.importPage(ctx, page)
		if err != nil {
			return out, err
		}
		local[wikidot.Normalize(page.Name)] = id
		byThread[page.ThreadID] = id
		out.Pages++
		out.Revisions += revisions

		files, err := im.importFiles(ctx, page, id, media)
		if err != nil {
			return out, err
		}
		out.Files += files

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

	delete(byThread, 0)
	if err := im.importForum(ctx, byThread, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (im *importer) importUsers(ctx context.Context) (map[int64]int64, error) {
	found, err := im.archive.Users()
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
	return im.db.EnsureWikidotUsers(ctx, list, time.Now().UTC())
}

func (im *importer) importPage(ctx context.Context, page Page) (int64, string, int, error) {
	sources, err := im.archive.Sources(im.slug, page)
	if err != nil {
		return 0, "", 0, err
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
		AuthorID:  localUser(im.users, oldest.Author),
	}

	// The archive lists the newest revision first and the history reads the
	// other way round.
	for i := len(page.Revisions) - 1; i >= 0; i-- {
		rev := page.Revisions[i]
		one := db.ImportRevision{
			Number:  rev.Number,
			UserID:  localUser(im.users, rev.Author),
			Comment: rev.Comment,
			At:      time.Unix(rev.Stamp, 0).UTC(),
			IsNew:   rev.IsNew(),
		}
		if source, ok := sources[rev.Number]; ok {
			one.Source = &source
		}
		write.Revisions = append(write.Revisions, one)
		if one.Source != nil {
			write.Indexed = *one.Source
		}
	}

	if im.opts.Votes {
		for _, vote := range page.Votings {
			user := localUser(im.users, vote.UserID)
			if user == nil {
				continue
			}
			write.Votes = append(write.Votes, db.ImportVote{UserID: *user, Rate: vote.Value})
		}
	}
	if len(page.Tags) > 0 {
		ids, err := im.db.EnsureTags(ctx, im.siteID, page.Tags, im.opts.ForceTags)
		if err != nil {
			return 0, "", 0, err
		}
		write.TagIDs = ids
	}

	id, media, err := im.db.ImportArticle(ctx, im.siteID, write)
	if err != nil {
		return 0, "", 0, err
	}
	return id, media, len(write.Revisions), nil
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
