package modules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("rate", renderRate) }

const (
	starColorRated   = "#f0ac00"
	starColorUnrated = "#4e6b6b"
)

func renderRate(env module.Env, _ map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", module.ErrNotPorted
	}
	var article *db.Article
	if env.Page != nil {
		article = env.Page.Article
	}
	if article == nil {
		return "", &module.Error{Message: env.Text("module-rate-no-page")}
	}

	mode, err := articleRatingMode(env, article.Category)
	if err != nil {
		return "", err
	}
	if mode != page.RatingModeUpDown && mode != page.RatingModeStars {
		return "", nil
	}
	stats, err := env.Data.VoteStats(article.ID)
	if err != nil {
		return "", err
	}
	rating := page.RatingOf(mode, stats)

	if mode == page.RatingModeUpDown {
		return upDownWidget(env, article.FullName(), rating), nil
	}

	// Django only reaches the vote lookup for a signed-in reader, so the
	// anonymous votes a null user id would match are never asked about.
	voted := false
	if env.User != nil {
		voted, err = env.Data.HasVoted(article.ID, &env.User.ID)
		if err != nil {
			return "", err
		}
	}
	return starsWidget(env, article.FullName(), rating, voted), nil
}

func articleRatingMode(env module.Env, category string) (string, error) {
	site, err := env.Data.SiteRatingMode()
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	own, err := env.Data.CategoryRatingMode(category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	return page.RatingMode(site, own), nil
}

func upDownWidget(env module.Env, pageID string, rating page.Rating) string {
	value, _ := rating.Value.(int)
	label := env.Text("module-rate-label")

	var b strings.Builder
	b.WriteString(`<div class="w-rate-module page-rate-widget-box" data-page-id="` + escape.HTML(pageID) + `">`)
	b.WriteString(`<span class="rate-points">` + label + `&nbsp;<span class="number prw54353">` +
		fmt.Sprintf("%+d", value) + `</span></span>`)
	b.WriteString(`<span class="rateup btn btn-default"><a title="` + env.Text("module-rate-up") + `" href="#">+</a></span>`)
	b.WriteString(`<span class="ratedown btn btn-default"><a title="` + env.Text("module-rate-down") + `" href="#">–</a></span>`)
	b.WriteString(`<span class="cancel btn btn-default"><a title="` + env.Text("module-rate-cancel") + `" href="#">x</a></span>`)
	b.WriteString(`</div>`)
	return b.String()
}

func starsWidget(env module.Env, pageID string, rating page.Rating, voted bool) string {
	average, _ := rating.Value.(float64)
	shown := "—"
	if rating.Votes != 0 {
		shown = fmt.Sprintf("%.1f", average)
	}
	color := starColorUnrated
	if voted {
		color = starColorRated
	}
	label := env.Text("module-rate-label")

	var b strings.Builder
	b.WriteString(`<div class="w-stars-rate-module" data-page-id="` + escape.HTML(pageID) + `">`)
	b.WriteString(`<div class="w-stars-rate-rating">` + label + `&nbsp;<span class="w-stars-rate-number">` + shown + `</span></div>`)
	b.WriteString(`<div class="w-stars-rate-control">`)
	b.WriteString(`<div class="w-stars-rate-stars-wrapper"><div class="w-stars-rate-stars-view" style="width: ` +
		strconv.Itoa(int(average*20)) + `%; --rated-var: ` + color + `"></div></div>`)
	b.WriteString(`<div class="w-stars-rate-cancel"></div>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="w-stars-rate-votes"><span class="w-stars-rate-number" title="` +
		env.Text("module-rate-votes") + `">` + strconv.Itoa(rating.Votes) + `</span>/<span class="w-stars-rate-popularity" title="` +
		env.Text("module-rate-popularity") + `">` + strconv.Itoa(rating.Popularity) + `</span>%</div>`)
	b.WriteString(`</div>`)
	return b.String()
}
