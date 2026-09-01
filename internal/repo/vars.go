package repo

import (
	"context"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

// VarSource backs the page variables of one request. The site is fixed for the
// whole render because the rating setting is resolved against the site the
// request arrived on.
type VarSource struct {
	ctx   context.Context
	db    *db.DB
	site  *db.Site
	forms *formLoader
}

var _ page.VarSource = (*VarSource)(nil)

func NewVarSource(ctx context.Context, d *db.DB, site *db.Site) *VarSource {
	return &VarSource{ctx: ctx, db: d, site: site, forms: newFormLoader(ctx, d)}
}

func (s *VarSource) LatestSource(articleID int64) (string, error) {
	return s.db.LatestSource(s.ctx, articleID)
}

func (s *VarSource) Authors(articleID int64) ([]db.User, error) {
	return s.db.ArticleAuthors(s.ctx, articleID)
}

func (s *VarSource) LatestEditor(articleID int64) (*db.User, error) {
	return s.db.LatestEditor(s.ctx, articleID)
}

func (s *VarSource) RevisionCount(articleID int64) (int, error) {
	return s.db.RevisionCount(s.ctx, articleID)
}

func (s *VarSource) Tags(articleID int64) ([]string, error) {
	return s.db.ArticleTags(s.ctx, articleID)
}

func (s *VarSource) VoteStats(articleID int64) (db.VoteStats, error) {
	return s.db.VoteStats(s.ctx, articleID)
}

func (s *VarSource) SiteRatingMode() (string, error) {
	return s.db.SiteRatingMode(s.ctx, s.site.ID)
}

func (s *VarSource) SiteName() string {
	if s.site == nil {
		return ""
	}
	return s.site.Slug
}

func (s *VarSource) CategoryRatingMode(category string) (string, error) {
	return s.db.CategoryRatingMode(s.ctx, category)
}

func (s *VarSource) HasVoted(articleID int64, userID *int64) (bool, error) {
	return s.db.HasVoted(s.ctx, articleID, userID)
}

func (s *VarSource) ArticleByID(id int64) (*db.Article, error) {
	return s.db.ArticleByID(s.ctx, id)
}

func (s *VarSource) CategoryForm(category string) (*form.Definition, error) {
	return s.forms.CategoryForm(category)
}
