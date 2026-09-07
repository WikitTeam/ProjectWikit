package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/changelog"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
)

const actionActivity = "activity"

const activityPerPage = 40

const (
	showEdits = "edits"
	showVotes = "votes"
	showPosts = "posts"
)

var activityTabs = []string{showEdits, showVotes, showPosts}

type userVoteRow struct {
	Title string
	Href  string
	Rate  string
	At    *time.Time
}

type userPostRow struct {
	Name      string
	Thread    string
	Article   string
	Href      string
	CreatedAt time.Time
}

func (h *Handler) userActivity(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	ctx := r.Context()
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.AdminUser(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}

	show := r.URL.Query().Get("show")
	if !contains(activityTabs, show) {
		show = showEdits
	}
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * activityPerPage

	data := map[string]any{
		"Show":   show,
		"Tabs":   activityTabs,
		"Page":   page,
		"Action": Prefix + userSlug + "/" + rest + "/" + actionActivity,
		"Back":   Prefix + userSlug + "/" + rest,
	}
	var rows int
	switch show {
	case showVotes:
		votes, err := h.userVotes(ctx, id, offset)
		if err != nil {
			return err
		}
		data["Votes"] = votes
		data["Dated"] = grantsFrom(ctx).Has(perms.ViewVotesTimestamp)
		rows = len(votes)
	case showPosts:
		posts, err := h.userPosts(ctx, id, offset)
		if err != nil {
			return err
		}
		data["Posts"] = posts
		rows = len(posts)
	default:
		edits, err := h.userEdits(ctx, loc, id, offset)
		if err != nil {
			return err
		}
		data["Edits"] = edits
		rows = len(edits)
	}
	data["More"] = rows == activityPerPage

	return h.page(w, r, loc, loc.T("admin.activity-of", "name", userLabel(row)), "user_activity.html", data)
}

func userLabel(row db.AdminUserRow) string {
	if row.Type == db.UserTypeWikidot && row.WikidotUsername != "" {
		return row.WikidotUsername
	}
	return row.Username
}

func (h *Handler) userEdits(ctx context.Context, loc *i18n.Localizer, id int64, offset int) ([]changeRow, error) {
	hidden, err := repo.HiddenCategories(ctx, h.deps.DB, auth.FromContext(ctx))
	if err != nil {
		return nil, err
	}
	found, err := h.deps.DB.SiteChanges(ctx, db.SiteChangeFilter{
		Hidden:  hidden,
		HasUser: true,
		UserIDs: []int64{id},
	}, offset, activityPerPage)
	if err != nil {
		return nil, err
	}
	users := func(want []int64) ([]db.User, error) { return h.deps.DB.UsersByIDs(ctx, want) }

	out := make([]changeRow, 0, len(found))
	for _, c := range found {
		article := db.Article{Category: c.ArticleCategory, Name: c.ArticleName, Title: c.ArticleTitle}
		row := changeRow{
			Title:     article.DisplayName(),
			Href:      "/" + article.FullName(),
			CreatedAt: c.CreatedAt,
		}
		entry, err := changelog.Of(loc, users, c)
		switch {
		case errors.Is(err, changelog.ErrUnreadable):
		case err != nil:
			return nil, err
		default:
			row.Flags = entry.Flags
			row.Comment = entry.Comment
		}
		out = append(out, row)
	}
	return out, nil
}

func (h *Handler) userVotes(ctx context.Context, id int64, offset int) ([]userVoteRow, error) {
	found, err := h.deps.DB.RatedBy(ctx, id, offset, activityPerPage)
	if err != nil {
		return nil, err
	}
	out := make([]userVoteRow, 0, len(found))
	for i := range found {
		one := &found[i]
		out = append(out, userVoteRow{
			Title: one.Article.DisplayName(),
			Href:  "/" + one.Article.FullName(),
			Rate:  strconv.FormatFloat(one.Rate, 'f', -1, 64),
			At:    one.VotedAt,
		})
	}
	return out, nil
}

func (h *Handler) userPosts(ctx context.Context, id int64, offset int) ([]userPostRow, error) {
	categories, err := h.deps.DB.ForumCategories(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(categories))
	for _, c := range categories {
		ids = append(ids, c.ID)
	}
	found, err := h.deps.DB.UserPosts(ctx, id, ids, true, offset, activityPerPage)
	if err != nil {
		return nil, err
	}

	out := make([]userPostRow, 0, len(found))
	for _, p := range found {
		row := userPostRow{
			Name:   p.Name,
			Thread: p.ThreadName,
			Href: "/forum/t-" + strconv.FormatInt(p.ThreadID, 10) +
				"#post-" + strconv.FormatInt(p.ID, 10),
			CreatedAt: p.CreatedAt,
		}
		if p.ArticleTitle != nil || p.ArticleName != nil {
			article := db.Article{}
			if p.ArticleCategory != nil {
				article.Category = *p.ArticleCategory
			}
			if p.ArticleName != nil {
				article.Name = *p.ArticleName
			}
			if p.ArticleTitle != nil {
				article.Title = *p.ArticleTitle
			}
			row.Article = article.DisplayName()
		}
		out = append(out, row)
	}
	return out, nil
}
