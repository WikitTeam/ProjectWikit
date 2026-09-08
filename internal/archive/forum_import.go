package archive

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/htmlsource"
)

const importedSection = "Imported"

// A thread carries no name a second run could recognise, so a site that already
// has a section is left alone rather than doubled.
func (im *importer) importForum(ctx context.Context, byArchiveThread map[int64]int64, out *Result) error {
	categories, err := im.archive.Categories(im.slug)
	if err != nil || len(categories) == 0 {
		return err
	}
	sections, err := im.db.AdminForumSections(ctx, im.siteID)
	if err != nil {
		return err
	}
	if len(sections) > 0 {
		report(im.opts, "the site already has a forum, leaving it alone")
		return nil
	}

	threads := make(map[int64][]Thread, len(categories))
	for _, category := range categories {
		found, err := im.archive.Threads(im.slug, category.ID)
		if err != nil {
			return err
		}
		threads[category.ID] = found
	}

	sectionID, err := im.db.ImportForumSection(ctx, im.siteID, importedSection, "")
	if err != nil {
		return err
	}

	for order, category := range categories {
		comments := 0
		for _, thread := range threads[category.ID] {
			if _, ok := byArchiveThread[thread.ID]; ok {
				comments++
			}
		}
		localCategory, err := im.db.ImportForumCategory(ctx, im.siteID, sectionID,
			category.Title, category.Description, order, comments*2 > len(threads[category.ID]))
		if err != nil {
			return err
		}
		out.Categories++

		for _, thread := range threads[category.ID] {
			posts, err := im.importThread(ctx, thread, localCategory, byArchiveThread)
			if err != nil {
				return err
			}
			out.Threads++
			out.Posts += posts
			if out.Threads%100 == 0 {
				report(im.opts, fmt.Sprintf("%d threads, %d posts", out.Threads, out.Posts))
			}
		}
	}
	return nil
}

func (im *importer) importThread(ctx context.Context, thread Thread, categoryID int64,
	byArchiveThread map[int64]int64) (int, error) {

	bodies, err := im.archive.PostBodies(im.slug, thread)
	if err != nil {
		return 0, err
	}
	posts, parents := im.flatten(thread.Posts, -1, nil, nil, bodies)

	write := db.ImportThread{
		CategoryID:  &categoryID,
		Name:        thread.Title,
		Description: thread.Description,
		AuthorID:    localUser(im.users, started(thread)),
		CreatedAt:   time.Unix(thread.Started, 0).UTC(),
		UpdatedAt:   time.Unix(thread.Last, 0).UTC(),
		Pinned:      thread.Sticky,
		Locked:      thread.Locked,
	}
	if article, ok := byArchiveThread[thread.ID]; ok {
		write.CategoryID = nil
		write.ArticleID = &article
	}
	return im.db.ImportForumThread(ctx, im.siteID, write, posts, parents)
}

func (im *importer) flatten(tree []Post, parent int, posts []db.ImportPost, parents []int,
	bodies map[string]string) ([]db.ImportPost, []int) {

	for _, post := range tree {
		posts = append(posts, db.ImportPost{
			Name:      post.Title,
			AuthorID:  localUser(im.users, post.Poster),
			CreatedAt: time.Unix(post.Stamp, 0).UTC(),
			Versions:  im.versions(post, bodies),
		})
		parents = append(parents, parent)
		posts, parents = im.flatten(post.Children, len(posts)-1, posts, parents, bodies)
	}
	return posts, parents
}

func (im *importer) versions(post Post, bodies map[string]string) []db.ImportPostVersion {
	if len(post.Revisions) == 0 {
		return []db.ImportPostVersion{{
			Source:   body(bodies, fmt.Sprintf("%d/%s", post.ID, postBodyFile)),
			AuthorID: localUser(im.users, post.Poster),
			At:       time.Unix(post.Stamp, 0).UTC(),
		}}
	}
	revisions := append([]PostRevision(nil), post.Revisions...)
	sort.SliceStable(revisions, func(i, j int) bool { return revisions[i].Stamp < revisions[j].Stamp })

	out := make([]db.ImportPostVersion, 0, len(revisions))
	for _, revision := range revisions {
		out = append(out, db.ImportPostVersion{
			Source:   body(bodies, fmt.Sprintf("%d/%d.html", post.ID, revision.ID)),
			AuthorID: localUser(im.users, revision.Author),
			At:       time.Unix(revision.Stamp, 0).UTC(),
		})
	}
	return out
}

// The archive pads a post with the layout whitespace it was served inside, and
// wikitext reads leading whitespace as markup.
func body(bodies map[string]string, key string) string {
	return strings.TrimSpace(htmlsource.Convert(bodies[key]))
}

func started(thread Thread) int64 {
	if thread.StartedUser != nil {
		return *thread.StartedUser
	}
	if len(thread.Posts) > 0 {
		return thread.Posts[0].Poster
	}
	return 0
}
