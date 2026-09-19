package archive

// Ratings and attachments left behind by the options name nobody who needs an
// account, so they are not counted.
func (a *Archive) usedUsers(slug string, pages []Page, opts Options) (map[int64]bool, error) {
	used := map[int64]bool{}
	for _, page := range pages {
		for _, rev := range page.Revisions {
			used[rev.Author] = true
		}
		if opts.Votes {
			for _, vote := range page.Votings {
				used[vote.UserID] = true
			}
		}
		if opts.Files != "" {
			for _, file := range page.Files {
				used[file.Author] = true
			}
		}
	}

	categories, err := a.Categories(slug)
	if err != nil {
		return nil, err
	}
	for _, category := range categories {
		threads, err := a.Threads(slug, category.ID)
		if err != nil {
			return nil, err
		}
		for _, thread := range threads {
			used[started(thread)] = true
			markPosters(used, thread.Posts)
		}
	}
	delete(used, 0)
	return used, nil
}

func markPosters(used map[int64]bool, posts []Post) {
	for _, post := range posts {
		used[post.Poster] = true
		for _, rev := range post.Revisions {
			used[rev.Author] = true
		}
		markPosters(used, post.Children)
	}
}
