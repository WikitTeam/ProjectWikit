package modules

import (
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() {
	module.RegisterWriteAPI("rate", "favourite", articleFavourite)
	module.RegisterWriteAPI("rate", "unfavourite", articleUnfavourite)
	module.RegisterAPI("rate", "get_favourites", articleFavourites)
}

func articleFavourite(env module.Env, params map[string]string) (wikijson.Object, error) {
	article, err := favouritableArticle(env, params)
	if err != nil {
		return nil, err
	}
	if err := env.Data.AddFavourite(article.ID, env.User.ID); err != nil {
		return nil, err
	}
	return favouriteState(env, article, true)
}

func articleUnfavourite(env module.Env, params map[string]string) (wikijson.Object, error) {
	article, err := favouritableArticle(env, params)
	if err != nil {
		return nil, err
	}
	if err := env.Data.RemoveFavourite(article.ID, env.User.ID); err != nil {
		return nil, err
	}
	return favouriteState(env, article, false)
}

func articleFavourites(env module.Env, params map[string]string) (wikijson.Object, error) {
	article, err := ratedFavouriteArticle(env, params)
	if err != nil {
		return nil, err
	}
	mine := false
	if env.User != nil {
		mine, err = env.Data.HasFavourited(article.ID, env.User.ID)
		if err != nil {
			return nil, err
		}
	}
	return favouriteState(env, article, mine)
}

// Only the count goes out. Who put a page on their list is nobody's business,
// not even the author's, so no query here can name them.
func favouriteState(env module.Env, article *db.Article, mine bool) (wikijson.Object, error) {
	count, err := env.Data.ArticleFavouriteCount(article.ID)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "pageId", Value: article.FullName()},
		{Key: "favourites", Value: count},
		{Key: "favourited", Value: mine},
	}, nil
}

func favouritableArticle(env module.Env, params map[string]string) (*db.Article, error) {
	if env.User == nil {
		return nil, &module.Error{Message: env.Text("favourite-signed-out")}
	}
	return ratedFavouriteArticle(env, params)
}

func ratedFavouriteArticle(env module.Env, params map[string]string) (*db.Article, error) {
	var article *db.Article
	if ref := params["pageid"]; ref != "" {
		found, err := env.Data.ArticleByRef(ref)
		if err != nil {
			return nil, err
		}
		article = found
	} else if env.Page != nil {
		article = env.Page.Article
	}
	if article == nil {
		return nil, &module.Error{Message: env.Text("module-rate-no-page")}
	}

	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	object, err := env.Data.ArticleObject(article, env.User)
	if err != nil {
		return nil, err
	}
	if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
		return nil, &module.Error{Message: env.Text("module-rate-no-page")}
	}
	return article, nil
}

func favouriteStarHTML(env module.Env, article *db.Article, count int, mine bool) string {
	state := "far"
	if mine {
		state = "fas"
	}
	return `<span class="w-favourite" data-page-id="` + escape.HTML(article.FullName()) +
		`" data-favourited="` + strconv.FormatBool(mine) +
		`" data-count="` + strconv.Itoa(count) + `">` +
		`<a href="javascript:;" class="favourite-toggle" title="` +
		escape.HTML(env.Text("favourite-toggle")) + `">` +
		`<i class="` + state + ` fa-star"></i></a>` +
		`<span class="favourite-count">` + strconv.Itoa(count) + `</span></span>`
}
