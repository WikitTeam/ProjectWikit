package modules

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() {
	module.RegisterAPI("rate", "get_rating", rateGetRating)
	module.RegisterAPI("rate", "get_votes", rateGetVotes)
	module.RegisterWriteAPI("rate", "rate", rateVote)
}

const migratedVoteDate = "migrated"

func rateGetRating(env module.Env, _ map[string]string) (wikijson.Object, error) {
	article, err := ratedArticle(env)
	if err != nil {
		return nil, err
	}
	rating, err := articleRating(env, article)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "pageId", Value: article.FullName()},
		{Key: "rating", Value: rating.Value},
		{Key: "voteCount", Value: rating.Votes},
		{Key: "popularity", Value: rating.Popularity},
		{Key: "ratingMode", Value: rating.Mode},
	}, nil
}

func rateGetVotes(env module.Env, _ map[string]string) (wikijson.Object, error) {
	article, err := ratedArticle(env)
	if err != nil {
		return nil, err
	}
	rating, err := articleRating(env, article)
	if err != nil {
		return nil, err
	}
	votes, err := env.Data.ArticleVotes(article.ID)
	if err != nil {
		return nil, err
	}

	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	withDate := perms.Resolve(subject, nil).Has(perms.ViewVotesTimestamp)

	out := make(wikijson.Array, 0, len(votes))
	for i := range votes {
		vote := votes[i]
		user, err := env.Data.UserJSON(&vote.User)
		if err != nil {
			return nil, err
		}
		var group, index any
		if vote.RoleID != nil {
			group, index = vote.GroupTitle, vote.GroupIndex
		}
		fields := wikijson.Object{
			{Key: "user", Value: user},
			{Key: "value", Value: vote.Rate},
			{Key: "visualGroup", Value: group},
			{Key: "visualGroupIndex", Value: index},
		}
		if withDate {
			fields = append(fields, wikijson.Field{Key: "date", Value: voteDate(vote.Date)})
		}
		out = append(out, fields)
	}
	return wikijson.Object{
		{Key: "pageId", Value: article.FullName()},
		{Key: "votes", Value: out},
		{Key: "rating", Value: rating.Value},
		{Key: "popularity", Value: rating.Popularity},
		{Key: "mode", Value: rating.Mode},
	}, nil
}

func rateVote(env module.Env, params map[string]string) (wikijson.Object, error) {
	article, err := ratedArticle(env)
	if err != nil {
		return nil, err
	}
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	object, err := env.Data.ArticleObject(article, env.User)
	if err != nil {
		return nil, err
	}
	if !perms.Resolve(subject, object).Has(perms.RateArticles) {
		return nil, &module.Error{Message: env.Text("module-rate-forbidden")}
	}

	raw, given := params["value"]
	if !given {
		return nil, &module.Error{Message: env.Text("module-rate-no-value")}
	}
	mode, err := ratingMode(env, article)
	if err != nil {
		return nil, err
	}
	value, err := voteValue(env, raw, mode)
	if err != nil {
		return nil, err
	}

	var userID int64
	if env.User != nil {
		userID = env.User.ID
	}
	role, err := env.Data.VoteGroupRole(userIDOrNil(env.User))
	if err != nil {
		return nil, err
	}
	old, err := env.Data.ReplaceVote(article.ID, userID, value, role)
	if err != nil {
		return nil, err
	}
	if err := logVote(env, article, old, value); err != nil {
		return nil, err
	}
	return rateGetRating(env, params)
}

// An empty value is the reader taking their vote back, which is what the null
// the browser sends arrives as.
func voteValue(env module.Env, raw, mode string) (*float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil, &module.Error{Message: env.Text("module-rate-bad-value", "value", raw)}
	}
	switch mode {
	case page.RatingModeUpDown:
		if value != -1 && value != 0 && value != 1 {
			return nil, &module.Error{Message: env.Text("module-rate-bad-value", "value", raw)}
		}
	case page.RatingModeStars:
		if value < 0 || value > 5 || math.Mod(value, 0.5) != 0 {
			return nil, &module.Error{Message: env.Text("module-rate-bad-value", "value", raw)}
		}
	}
	return &value, nil
}

func logVote(env module.Env, article *db.Article, old *db.Vote, value *float64) error {
	var was, now any
	if old != nil {
		was = old.Rate
	}
	if value != nil {
		now = *value
	}
	meta, err := wikijson.Marshal(wikijson.Object{
		{Key: "article", Value: article.FullName()},
		{Key: "old_vote", Value: was},
		{Key: "new_vote", Value: now},
		{Key: "is_new", Value: value != nil && old == nil},
		{Key: "is_change", Value: value != nil && old != nil},
		{Key: "is_remove", Value: value == nil && old != nil},
	})
	if err != nil {
		return err
	}
	return env.Data.AddActionLog(env.User, db.ActionVote, meta)
}

func ratedArticle(env module.Env) (*db.Article, error) {
	if env.Page == nil || env.Page.Article == nil {
		return nil, &module.Error{Message: env.Text("module-rate-no-page")}
	}
	return env.Page.Article, nil
}

func articleRating(env module.Env, article *db.Article) (page.Rating, error) {
	mode, err := ratingMode(env, article)
	if err != nil {
		return page.Rating{}, err
	}
	if mode == page.RatingModeDisabled {
		return page.RatingOf(mode, db.VoteStats{}), nil
	}
	stats, err := env.Data.VoteStats(article.ID)
	if err != nil {
		return page.Rating{}, err
	}
	return page.RatingOf(mode, stats), nil
}

func ratingMode(env module.Env, article *db.Article) (string, error) {
	site, err := env.Data.SiteRatingMode()
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	category, err := env.Data.CategoryRatingMode(article.Category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	return page.RatingMode(site, category), nil
}

// A vote with no timestamp came in with the imported site, and the word stands
// in for the date the frontend would otherwise print.
func voteDate(at *time.Time) string {
	if at == nil {
		return migratedVoteDate
	}
	return at.UTC().Format("2006-01-02T15:04:05.999999-07:00")
}

func userIDOrNil(u *db.User) *int64 {
	if u == nil {
		return nil
	}
	return &u.ID
}
