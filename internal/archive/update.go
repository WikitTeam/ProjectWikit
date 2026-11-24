package archive

import (
	"context"
	"errors"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type UpdatePlan struct {
	New     int
	Current int
	Update  []string
	Edited  []string
}

type UpdateResult struct {
	Pages      int
	Revisions  int
	Edited     []string
	Votes      int64
	Files      int
	Categories int
	Threads    int
	Posts      int
	Renumbered []db.Renumbered
}

type pageState int

const (
	stateCurrent pageState = iota
	stateNewer
	stateEdited
)

func PlanUpdate(ctx context.Context, d *db.DB, siteID int64, a *Archive, slug string) (UpdatePlan, error) {
	var plan UpdatePlan
	pages, err := a.Pages(slug)
	if err != nil {
		return plan, err
	}
	for _, page := range pages {
		article, err := d.ArticleByName(ctx, siteID, page.Name)
		if errors.Is(err, db.ErrNotFound) {
			plan.New++
			continue
		}
		if err != nil {
			return plan, err
		}
		state, _, err := stateOf(ctx, d, article.ID, page)
		if err != nil {
			return plan, err
		}
		switch state {
		case stateNewer:
			plan.Update = append(plan.Update, page.Name)
		case stateEdited:
			plan.Edited = append(plan.Edited, page.Name)
		default:
			plan.Current++
		}
	}
	return plan, nil
}

func stateOf(ctx context.Context, d *db.DB, articleID int64, page Page) (pageState, []Revision, error) {
	head, err := d.ArticleHead(ctx, articleID)
	if errors.Is(err, db.ErrNotFound) {
		return stateEdited, nil, nil
	}
	if err != nil {
		return stateCurrent, nil, err
	}
	matched := false
	var newer []Revision
	for _, rev := range page.Revisions {
		if rev.Number == head.RevNumber && time.Unix(rev.Stamp, 0).UTC().Equal(head.At) {
			matched = true
		}
		if rev.Number > head.RevNumber {
			newer = append(newer, rev)
		}
	}
	switch {
	case !matched:
		return stateEdited, nil, nil
	case len(newer) == 0:
		return stateCurrent, nil, nil
	}
	return stateNewer, newer, nil
}

func ApplyUpdate(ctx context.Context, d *db.DB, siteID int64, a *Archive, slug string, opts Options) (UpdateResult, error) {
	var out UpdateResult
	im := &importer{db: d, archive: a, slug: slug, siteID: siteID, opts: opts}

	pages, err := a.Pages(slug)
	if err != nil {
		return out, err
	}
	if im.users, err = im.importUsers(ctx, pages); err != nil {
		return out, err
	}

	byThread := map[int64]int64{}
	for _, page := range pages {
		article, err := d.ArticleByName(ctx, siteID, page.Name)
		if errors.Is(err, db.ErrNotFound) {
			continue
		}
		if err != nil {
			return out, err
		}
		byThread[page.ThreadID] = article.ID

		state, newer, err := stateOf(ctx, d, article.ID, page)
		if err != nil {
			return out, err
		}
		switch state {
		case stateEdited:
			out.Edited = append(out.Edited, page.Name)
		case stateNewer:
			n, err := im.updatePage(ctx, page, article.ID, newer)
			if err != nil {
				return out, err
			}
			out.Pages++
			out.Revisions += n
		}

		votes, err := im.addVotes(ctx, page, article.ID)
		if err != nil {
			return out, err
		}
		out.Votes += votes
		files, err := im.addFiles(ctx, page, article)
		if err != nil {
			return out, err
		}
		out.Files += files
	}
	delete(byThread, 0)
	err = im.updateForum(ctx, byThread, &out)
	out.Renumbered = im.renumbered
	return out, err
}

func (im *importer) updatePage(ctx context.Context, page Page, articleID int64, newer []Revision) (int, error) {
	sources, err := im.archive.Sources(im.slug, page)
	if err != nil {
		return 0, err
	}
	u := db.ImportUpdate{
		Title:     page.Title,
		Locked:    page.Locked,
		UpdatedAt: time.Unix(page.Revisions[0].Stamp, 0).UTC(),
	}
	for i := len(newer) - 1; i >= 0; i-- {
		rev := newer[i]
		one := db.ImportRevision{
			Number:  rev.Number,
			UserID:  localUser(im.users, rev.Author),
			Comment: rev.Comment,
			At:      time.Unix(rev.Stamp, 0).UTC(),
			IsNew:   rev.IsNew(),
		}
		if source, ok := sources[rev.Number]; ok {
			one.Source = &source
			u.Indexed = source
		}
		u.Revisions = append(u.Revisions, one)
	}
	if im.opts.Tags {
		ids, err := im.db.EnsureTags(ctx, im.siteID, page.Tags)
		if err != nil {
			return 0, err
		}
		u.ReplaceTags = true
		u.TagIDs = ids
	}
	return len(u.Revisions), im.db.UpdateImportedArticle(ctx, im.siteID, articleID, u)
}

func (im *importer) addVotes(ctx context.Context, page Page, articleID int64) (int64, error) {
	if !im.opts.Votes {
		return 0, nil
	}
	var added int64
	for _, vote := range page.Votings {
		user := localUser(im.users, vote.UserID)
		if user == nil {
			continue
		}
		n, err := im.db.AddImportedVote(ctx, articleID, *user, vote.Value)
		if err != nil {
			return added, err
		}
		added += n
	}
	return added, nil
}

func (im *importer) addFiles(ctx context.Context, page Page, article *db.Article) (int, error) {
	if im.opts.Files == nil || len(page.Files) == 0 {
		return 0, nil
	}
	have, err := im.db.ArticleFileNames(ctx, article.ID)
	if err != nil {
		return 0, err
	}
	fresh := page
	fresh.Files = nil
	for _, file := range page.Files {
		if !have[file.Name] {
			fresh.Files = append(fresh.Files, file)
		}
	}
	if len(fresh.Files) == 0 {
		return 0, nil
	}
	return im.importFiles(ctx, fresh, article.ID, article.MediaName)
}

func (im *importer) updateForum(ctx context.Context, byArchiveThread map[int64]int64, out *UpdateResult) (err error) {
	categories, err := im.archive.Categories(im.slug)
	if err != nil || len(categories) == 0 {
		return err
	}
	sectionID, err := im.db.ImportedSection(ctx, im.siteID, importedSection)
	if errors.Is(err, db.ErrNotFound) {
		report(im.opts, "the site's forum was not imported, leaving it alone")
		return nil
	}
	if err != nil {
		return err
	}
	defer func() {
		if bumpErr := im.db.BumpForumSequences(context.WithoutCancel(ctx)); err == nil {
			err = bumpErr
		}
	}()

	for order, category := range categories {
		threads, err := im.archive.Threads(im.slug, category.ID)
		if err != nil {
			return err
		}
		localCategory := int64(0)
		for _, thread := range threads {
			threadID, err := im.findThread(ctx, category, thread, byArchiveThread)
			if err != nil && !errors.Is(err, db.ErrNotFound) {
				return err
			}
			if err == nil {
				added, err := im.appendPosts(ctx, thread, threadID)
				if err != nil {
					return err
				}
				out.Posts += added
				continue
			}
			if _, ok := byArchiveThread[thread.ID]; !ok && localCategory == 0 {
				if localCategory, err = im.ensureCategory(ctx, sectionID, category, order, threads, byArchiveThread, out); err != nil {
					return err
				}
			}
			posts, err := im.importThread(ctx, thread, localCategory, byArchiveThread)
			if err != nil {
				return err
			}
			out.Threads++
			out.Posts += posts
		}
	}
	return nil
}

func (im *importer) ensureCategory(ctx context.Context, sectionID int64, category Category, order int,
	threads []Thread, byArchiveThread map[int64]int64, out *UpdateResult) (int64, error) {

	id, err := im.db.ImportedCategory(ctx, im.siteID, sectionID, category.Title)
	if !errors.Is(err, db.ErrNotFound) {
		return id, err
	}
	comments := 0
	for _, thread := range threads {
		if _, ok := byArchiveThread[thread.ID]; ok {
			comments++
		}
	}
	id, moved, err := im.db.ImportForumCategory(ctx, im.siteID, sectionID, category.ID, category.Title, category.Description,
		order, comments*2 > len(threads))
	if err != nil {
		return 0, err
	}
	im.moved(moved)
	out.Categories++
	return id, nil
}

func (im *importer) appendPosts(ctx context.Context, thread Thread, threadID int64) (int, error) {
	bodies, err := im.archive.PostBodies(im.slug, thread)
	if err != nil {
		return 0, err
	}
	posts, parents := im.flatten(thread.Posts, -1, nil, nil, bodies)
	stored, err := im.db.ThreadPostKeys(ctx, threadID)
	if err != nil {
		return 0, err
	}
	used := make([]bool, len(stored))
	ids := make([]int64, len(posts))
	added := 0
	for i, post := range posts {
		if j := matchPost(stored, used, post); j >= 0 {
			used[j] = true
			ids[i] = stored[j].ID
			continue
		}
		var replyTo *int64
		if parents[i] >= 0 && ids[parents[i]] != 0 {
			parent := ids[parents[i]]
			replyTo = &parent
		}
		id, moved, err := im.db.AppendImportedPost(ctx, threadID, post, replyTo)
		if err != nil {
			return added, err
		}
		im.moved(moved)
		ids[i] = id
		added++
	}
	return added, nil
}

func matchPost(stored []db.PostKey, used []bool, post db.ImportPost) int {
	for j, key := range stored {
		if used[j] || !key.At.Equal(post.CreatedAt) {
			continue
		}
		if key.AuthorID == nil || post.AuthorID == nil || *key.AuthorID == *post.AuthorID {
			return j
		}
	}
	return -1
}
