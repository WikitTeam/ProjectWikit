package modules

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() { module.RegisterAPI("search", "search", searchAPI) }

const (
	searchLimit = 20

	// What a result shows is the text around the word that was searched for,
	// not the head of the page.
	searchExcerptPad = 20
	searchExcerptLen = 160
	searchDateLayout = "2006-01-02"
)

var searchSpaces = regexp.MustCompile(`\s+`)

func searchAPI(env module.Env, params map[string]string) (wikijson.Object, error) {
	query := strings.TrimSpace(params["q"])
	author := strings.TrimSpace(params["author"])
	category := strings.TrimSpace(params["category"])
	tags := searchTags(params["tags"])
	from := searchDate(params["datefrom"], false)
	to := searchDate(params["dateto"], true)

	offset, err := strconv.Atoi(strings.TrimSpace(params["offset"]))
	if err != nil || offset < 0 {
		offset = 0
	}

	empty := wikijson.Object{
		{Key: "results", Value: wikijson.Array{}},
		{Key: "hasMore", Value: false},
		{Key: "total", Value: 0},
	}
	if query == "" && author == "" && category == "" && len(tags) == 0 && from == nil && to == nil {
		return empty, nil
	}

	filter, ok, err := searchFilter(env, query, author, category, tags, from, to)
	if err != nil {
		return nil, err
	}
	if !ok {
		return empty, nil
	}

	total, err := env.Data.SearchCount(filter)
	if err != nil {
		return nil, err
	}
	hits, err := env.Data.SearchArticles(filter, offset, searchLimit+1)
	if err != nil {
		return nil, err
	}
	hasMore := len(hits) > searchLimit
	if hasMore {
		hits = hits[:searchLimit]
	}

	results, err := searchResults(env, hits, filter, splitWords(query))
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "results", Value: results},
		{Key: "hasMore", Value: hasMore},
		{Key: "total", Value: total},
	}, nil
}

// A filter that cannot match anything comes back false rather than empty, since
// a tag nobody uses is not the same question as no tag at all.
func searchFilter(env module.Env, query, author, category string, tags []string,
	from, to *time.Time) (db.SearchFilter, bool, error) {

	hidden, err := env.Data.HiddenCategories(env.User)
	if err != nil {
		return db.SearchFilter{}, false, err
	}
	filter := db.SearchFilter{
		Words:    splitWords(query),
		Category: category,
		From:     from,
		To:       to,
		Hidden:   hidden,
	}

	if author != "" {
		found, err := searchAuthor(env, author)
		if err != nil {
			return db.SearchFilter{}, false, err
		}
		if found == nil {
			return db.SearchFilter{}, false, nil
		}
		filter.AuthorID = &found.ID
	}

	for _, name := range tags {
		if excluded, ok := strings.CutPrefix(name, "-"); ok {
			ids, err := searchTagIDs(env, excluded)
			if err != nil {
				return db.SearchFilter{}, false, err
			}
			filter.Exclude = append(filter.Exclude, ids...)
			continue
		}
		ids, err := searchTagIDs(env, name)
		if err != nil {
			return db.SearchFilter{}, false, err
		}
		if len(ids) == 0 {
			return db.SearchFilter{}, false, nil
		}
		filter.Include = append(filter.Include, ids)
	}
	return filter, true, nil
}

func searchTagIDs(env module.Env, name string) ([]int64, error) {
	category, tag := "", name
	if strings.Contains(name, ":") {
		category, tag = wikidot.Split(name)
	}
	return env.Data.TagIDsByFullName(category, tag)
}

func searchAuthor(env module.Env, name string) (*db.User, error) {
	canon := wikidot.CanonicalizeUsername(name)
	user, err := env.Data.UserByUsername(canon)
	if err == nil && user != nil {
		return user, nil
	}
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	user, err = env.Data.UserByWikidotName(canon)
	if err == nil && user != nil {
		return user, nil
	}
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	user, err = env.Data.UserByDisplayName(name)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	return user, err
}

func searchResults(env module.Env, hits []db.SearchHit, filter db.SearchFilter, words []string) (wikijson.Array, error) {
	ids := make([]int64, 0, len(hits))
	for i := range hits {
		ids = append(ids, hits[i].Article.ID)
	}
	authors, err := env.Data.AuthorsOfArticles(ids)
	if err != nil {
		return nil, err
	}
	votes, err := env.Data.VoteStatsOfArticles(ids)
	if err != nil {
		return nil, err
	}
	comments, err := env.Data.CommentCountsOfArticles(ids)
	if err != nil {
		return nil, err
	}
	tags, err := env.Data.TagsOfArticles(ids)
	if err != nil {
		return nil, err
	}
	wanted := map[int64]bool{}
	for _, group := range filter.Include {
		for _, id := range group {
			wanted[id] = true
		}
	}

	out := make(wikijson.Array, 0, len(hits))
	for i := range hits {
		article := &hits[i].Article
		rating, err := searchRating(env, article, votes[article.ID])
		if err != nil {
			return nil, err
		}
		out = append(out, wikijson.Object{
			{Key: "title", Value: firstNonBlank(article.Title, article.Name)},
			{Key: "url", Value: "/" + article.FullName()},
			{Key: "excerpt", Value: searchExcerpt(hits[i].Plaintext, words)},
			{Key: "words", Value: words},
			{Key: "author", Value: searchAuthorField(authors[article.ID])},
			{Key: "tags", Value: shownTags(tags[article.ID], wanted)},
			{Key: "comments", Value: comments[article.ID]},
			{Key: "createdAt", Value: searchTime(article.CreatedAt)},
			{Key: "updatedAt", Value: searchTime(article.UpdatedAt)},
			{Key: "rating", Value: rating},
		})
	}
	return out, nil
}

func searchRating(env module.Env, article *db.Article, stats db.VoteStats) (any, error) {
	siteMode, err := env.Data.SiteRatingMode()
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	categoryMode, err := env.Data.CategoryRatingMode(article.Category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	mode := page.RatingMode(siteMode, categoryMode)
	rating := page.RatingOf(mode, stats)

	switch mode {
	case page.RatingModeDisabled:
		return nil, nil
	case page.RatingModeStars:
		if rating.Votes == 0 {
			return "—", nil
		}
		return fmt.Sprintf("%.1f", rating.Value), nil
	case page.RatingModeUpDown:
		return fmt.Sprintf("%+d", rating.Value), nil
	}
	return fmt.Sprintf("%d", rating.Value), nil
}

func searchAuthorField(authors []db.User) any {
	if len(authors) == 0 {
		return nil
	}
	author := authors[0]
	return wikijson.Object{
		{Key: "name", Value: author.DisplayLabel()},
		{Key: "url", Value: "/-/users/" + urlName(author)},
	}
}

func urlName(u db.User) string {
	if u.Type == printuser.TypeWikidot {
		return firstNonBlank(u.WikidotUsername, u.Username)
	}
	return u.Username
}

func shownTags(tags []db.ArticleTag, wanted map[int64]bool) []string {
	out := []string{}
	if len(wanted) == 0 {
		return out
	}
	for _, tag := range tags {
		if wanted[tag.ID] {
			out = append(out, tag.FullName())
		}
	}
	return out
}

func searchExcerpt(plaintext string, words []string) string {
	_, body, found := strings.Cut(plaintext, "\n\n")
	if !found {
		body = plaintext
	}
	body = strings.TrimSpace(searchSpaces.ReplaceAllString(body, " "))
	if body == "" {
		return ""
	}

	runes := []rune(body)
	start := 0
	end := min(len(runes), searchExcerptLen)
	if at, length, ok := firstWord(body, words); ok {
		start = max(0, at-searchExcerptPad)
		end = min(len(runes), at+length+searchExcerptPad)
	}

	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
		snippet += "…"
	}
	return snippet
}

// The position is counted in characters rather than bytes, since the excerpt is
// cut in characters and a page of Chinese would otherwise be sliced mid-letter.
func firstWord(body string, words []string) (at, length int, ok bool) {
	lower := strings.ToLower(body)
	best := -1
	size := 0
	for _, word := range words {
		if word == "" {
			continue
		}
		index := strings.Index(lower, strings.ToLower(word))
		if index < 0 {
			continue
		}
		if best < 0 || index < best {
			best, size = index, len([]rune(word))
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	return len([]rune(body[:best])), size, true
}

func searchTags(raw string) []string {
	var out []string
	for _, name := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func splitWords(query string) []string {
	out := []string{}
	for _, word := range searchSpaces.Split(strings.TrimSpace(query), -1) {
		if word != "" {
			out = append(out, word)
		}
	}
	return out
}

func searchDate(raw string, endOfDay bool) *time.Time {
	value, err := time.ParseInLocation(searchDateLayout, strings.TrimSpace(raw), time.UTC)
	if err != nil {
		return nil
	}
	if endOfDay {
		value = value.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	return &value
}

func searchTime(at time.Time) any {
	if at.IsZero() {
		return nil
	}
	return at.UTC().Format("2006-01-02T15:04:05.999999-07:00")
}
