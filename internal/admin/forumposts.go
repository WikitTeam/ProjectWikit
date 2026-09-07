package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const recentPostSlug = "forum-posts"

const recentPostsPerPage = 40

func init() {
	register(screen{slug: recentPostSlug, label: "admin.forum-posts", need: perms.ManageForum, serve: (*Handler).forumPosts})
}

type recentPostRow struct {
	ID        int64
	Name      string
	Author    string
	Thread    string
	Category  string
	Href      string
	CreatedAt time.Time
}

func (h *Handler) forumPosts(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	ctx := r.Context()
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	comments := r.URL.Query().Get("comments") != "off"

	rows, err := h.recentPosts(ctx, comments, (page-1)*recentPostsPerPage, recentPostsPerPage)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.forum-posts"), "forum_posts.html", map[string]any{
		"Posts":    rows,
		"Comments": comments,
		"Page":     page,
		"More":     len(rows) == recentPostsPerPage,
		"Action":   Prefix + recentPostSlug + "/",
	})
}

func (h *Handler) recentPosts(ctx context.Context, comments bool, offset, limit int) ([]recentPostRow, error) {
	categories, err := h.deps.DB.ForumCategories(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(categories))
	named := make(map[int64]string, len(categories))
	for _, c := range categories {
		ids = append(ids, c.ID)
		named[c.ID] = c.Name
	}

	posts, err := h.deps.DB.RecentPosts(ctx, ids, comments, offset, limit)
	if err != nil {
		return nil, err
	}
	var authors []int64
	for _, p := range posts {
		if p.AuthorID != nil {
			authors = append(authors, *p.AuthorID)
		}
	}
	names, err := h.userNames(ctx, authors)
	if err != nil {
		return nil, err
	}

	out := make([]recentPostRow, 0, len(posts))
	for _, p := range posts {
		row := recentPostRow{
			ID:     p.ID,
			Name:   p.Name,
			Thread: p.ThreadName,
			Href: "/forum/t-" + strconv.FormatInt(p.ThreadID, 10) +
				"#post-" + strconv.FormatInt(p.ID, 10),
			CreatedAt: p.CreatedAt,
		}
		if p.AuthorID != nil {
			row.Author = names[*p.AuthorID]
		}
		if p.ThreadCategoryID != nil {
			row.Category = named[*p.ThreadCategoryID]
		}
		out = append(out, row)
	}
	return out, nil
}
