package repo

import (
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

type moduleData struct{ repo *Repository }

func (m moduleData) TagCategory(slug string) (db.TagCategory, error) {
	return m.repo.db.TagCategoryBySlug(m.repo.ctx, slug)
}

func (m moduleData) TagArticles(categorySlug, name string, hidden []string) ([]db.Article, error) {
	return m.repo.db.ArticlesByTag(m.repo.ctx, categorySlug, name, hidden)
}

// HiddenCategories asks the same question once per category, the way the page
// list does. There is no query for it because permissions are resolved in Go.
func (m moduleData) HiddenCategories(user *db.User) ([]string, error) {
	names, err := m.repo.db.CategoryNames(m.repo.ctx)
	if err != nil {
		return nil, err
	}
	resolver := NewPerms(m.repo.ctx, m.repo.db)
	subject, err := resolver.Subject(user, time.Now())
	if err != nil {
		return nil, err
	}

	var hidden []string
	for _, name := range names {
		object, err := resolver.Category(name)
		if err != nil {
			return nil, err
		}
		if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
			hidden = append(hidden, name)
		}
	}
	return hidden, nil
}

func (m moduleData) TagIDsByName(categorySlug, name string) ([]int64, error) {
	return m.repo.db.TagIDsByName(m.repo.ctx, categorySlug, name)
}

func (m moduleData) ArticleTagIDs(articleID int64) ([]int64, error) {
	return m.repo.db.ArticleTagIDs(m.repo.ctx, articleID)
}

func (m moduleData) ArticleByRef(ref string) (*db.Article, error) {
	return m.repo.db.ArticleByName(m.repo.ctx, ref)
}

func (m moduleData) UserByUsername(name string) (*db.User, error) {
	return m.repo.db.UserByUsername(m.repo.ctx, name)
}

func (m moduleData) UserByWikidotName(name string) (*db.User, error) {
	return m.repo.db.UserByWikidotName(m.repo.ctx, name)
}

func (m moduleData) SiteRatingMode() (string, error) {
	if m.repo.opts.Site == nil {
		return "", db.ErrNotFound
	}
	return m.repo.db.SiteRatingMode(m.repo.ctx, m.repo.opts.Site.ID)
}

func (m moduleData) CategoryRatingMode(category string) (string, error) {
	return m.repo.db.CategoryRatingMode(m.repo.ctx, category)
}

func (m moduleData) VoteStats(articleID int64) (db.VoteStats, error) {
	return m.repo.db.VoteStats(m.repo.ctx, articleID)
}

func (m moduleData) HasVoted(articleID int64, userID *int64) (bool, error) {
	return m.repo.db.HasVoted(m.repo.ctx, articleID, userID)
}

func (m moduleData) ListArticles(f db.ListFilter, offset int, limit *int) ([]db.Article, error) {
	return m.repo.db.ListArticles(m.repo.ctx, f, offset, limit)
}

func (m moduleData) CountArticles(f db.ListFilter, offset int, limit *int) (int, error) {
	return m.repo.db.CountArticles(m.repo.ctx, f, offset, limit)
}

func (m moduleData) ArticleByID(id int64) (*db.Article, error) {
	return m.repo.db.ArticleByID(m.repo.ctx, id)
}

func (m moduleData) ArticleAuthors(articleID int64) ([]db.User, error) {
	return m.repo.db.ArticleAuthors(m.repo.ctx, articleID)
}

func (m moduleData) TagCloud(limit *int) ([]db.CloudTag, error) {
	return m.repo.db.TagCloud(m.repo.ctx, limit)
}

func (m moduleData) RenderUser(u db.User) (string, error) {
	return m.repo.renderUser(&u, printuser.Options{Avatar: true, Hover: true})
}

// A post with no author is one the site itself made, which is what the system
// chip stands for.
func (m moduleData) RenderUserByID(id *int64) (string, error) {
	if id == nil {
		return m.repo.users.System(printuser.Options{Hover: true}), nil
	}
	user, err := m.repo.db.UserByID(m.repo.ctx, *id)
	if err != nil {
		return "", err
	}
	return m.repo.renderUser(user, printuser.Options{Avatar: true, Hover: true})
}

func (m moduleData) ForumSections() ([]db.ForumSection, error) {
	return m.repo.db.ForumSections(m.repo.ctx)
}

func (m moduleData) ForumSection(id int64) (*db.ForumSection, error) {
	return m.repo.db.ForumSection(m.repo.ctx, id)
}

func (m moduleData) ForumCategories() ([]db.ForumCategory, error) {
	return m.repo.db.ForumCategories(m.repo.ctx)
}

func (m moduleData) ForumCategoryCounts(categoryID int64) (db.ForumCounts, error) {
	return m.repo.db.ForumCategoryCounts(m.repo.ctx, categoryID)
}

func (m moduleData) ForumCommentCounts() (db.ForumCounts, error) {
	return m.repo.db.ForumCommentCounts(m.repo.ctx)
}

func (m moduleData) ForumCategoryLastPost(categoryID int64) (*db.ForumLastPost, error) {
	return m.repo.db.ForumCategoryLastPost(m.repo.ctx, categoryID)
}

func (m moduleData) ForumCommentLastPost() (*db.ForumLastPost, error) {
	return m.repo.db.ForumCommentLastPost(m.repo.ctx)
}

func (m moduleData) Subject(user *db.User) (perms.Subject, error) {
	return NewPerms(m.repo.ctx, m.repo.db).Subject(user, time.Now())
}

func (m moduleData) ForumSectionObject(s *db.ForumSection) *perms.Object {
	return NewPerms(m.repo.ctx, m.repo.db).ForumSection(s)
}

func (m moduleData) ForumCategory(id int64) (*db.ForumCategory, error) {
	return m.repo.db.ForumCategory(m.repo.ctx, id)
}

func (m moduleData) ForumThreads(categoryID int64, sort db.ForumThreadSort, offset, limit int) ([]db.ForumThread, error) {
	return m.repo.db.ForumThreads(m.repo.ctx, categoryID, sort, offset, limit)
}

func (m moduleData) ForumCommentThreads(sort db.ForumThreadSort, offset, limit int) ([]db.ForumThread, error) {
	return m.repo.db.ForumCommentThreads(m.repo.ctx, sort, offset, limit)
}

func (m moduleData) ForumThreadPostCount(threadID int64) (int, error) {
	return m.repo.db.ForumThreadPostCount(m.repo.ctx, threadID)
}

func (m moduleData) ForumThreadLastPost(threadID int64, count int) (*db.ForumPost, error) {
	return m.repo.db.ForumThreadLastPost(m.repo.ctx, threadID, count)
}

func (m moduleData) ForumThread(id int64) (*db.ForumThread, error) {
	return m.repo.db.ForumThread(m.repo.ctx, id)
}

func (m moduleData) ForumRootPostCount(threadID int64) (int, error) {
	return m.repo.db.ForumRootPostCount(m.repo.ctx, threadID)
}

func (m moduleData) ForumRootPosts(threadID int64, offset, limit int) ([]db.ForumThreadPost, error) {
	return m.repo.db.ForumRootPosts(m.repo.ctx, threadID, offset, limit)
}

func (m moduleData) ForumRootPostIDs(threadID int64) ([]int64, error) {
	return m.repo.db.ForumRootPostIDs(m.repo.ctx, threadID)
}

func (m moduleData) ForumPostReplies(postID int64) ([]db.ForumThreadPost, error) {
	return m.repo.db.ForumPostReplies(m.repo.ctx, postID)
}

func (m moduleData) ForumPost(id int64) (*db.ForumThreadPost, error) {
	return m.repo.db.ForumPost(m.repo.ctx, id)
}

func (m moduleData) ForumPostContents(postIDs []int64) (map[int64]db.ForumPostContent, error) {
	return m.repo.db.ForumPostContents(m.repo.ctx, postIDs)
}

func (m moduleData) RecentPostCount(categoryIDs []int64, comments bool) (int, error) {
	return m.repo.db.RecentPostCount(m.repo.ctx, categoryIDs, comments)
}

func (m moduleData) RecentPosts(categoryIDs []int64, comments bool, offset, limit int) ([]db.RecentPost, error) {
	return m.repo.db.RecentPosts(m.repo.ctx, categoryIDs, comments, offset, limit)
}

func (m moduleData) UsernamesLower() (map[string]bool, error) {
	return m.repo.db.UsernamesLower(m.repo.ctx)
}

func (m moduleData) VoteByUser(articleID int64, userID *int64) (float64, bool, error) {
	return m.repo.db.VoteByUser(m.repo.ctx, articleID, userID)
}

func (m moduleData) UserByID(id int64) (*db.User, error) {
	return m.repo.db.UserByID(m.repo.ctx, id)
}

func (m moduleData) ArticleHasAuthor(articleID, userID int64) (bool, error) {
	return m.repo.db.ArticleHasAuthor(m.repo.ctx, articleID, userID)
}

func (m moduleData) RolesByUser(id int64) ([]roles.Role, error) {
	return m.repo.db.RolesByUser(m.repo.ctx, id)
}

func (m moduleData) ForumThreadObject(t *db.ForumThread, u *db.User) (*perms.Object, error) {
	return NewPerms(m.repo.ctx, m.repo.db).ForumThread(t, u)
}

func (m moduleData) ForumPostObject(p *db.ForumThreadPost, thread *perms.Object, u *db.User) *perms.Object {
	return NewPerms(m.repo.ctx, m.repo.db).ForumPost(p, thread, u)
}

func (m moduleData) SiteChanges(f db.SiteChangeFilter, offset, limit int) ([]db.SiteChange, error) {
	return m.repo.db.SiteChanges(m.repo.ctx, f, offset, limit)
}

func (m moduleData) SiteChangeCount(f db.SiteChangeFilter) (int, error) {
	return m.repo.db.SiteChangeCount(m.repo.ctx, f)
}

func (m moduleData) ArticleCategories(hidden []string) ([]string, error) {
	return m.repo.db.ArticleCategories(m.repo.ctx, hidden)
}

func (m moduleData) UserIDsByName(name string, partial bool) ([]int64, error) {
	return m.repo.db.UserIDsByName(m.repo.ctx, name, partial)
}

func (m moduleData) UsersByIDs(ids []int64) ([]db.User, error) {
	return m.repo.db.UsersByIDs(m.repo.ctx, ids)
}
