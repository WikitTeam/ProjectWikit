// Package listpages answers [[module ListPages]].
package listpages

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type Source interface {
	TagIDsByName(categorySlug, name string) ([]int64, error)
	ArticleTagIDs(articleID int64) ([]int64, error)
	ArticleByRef(ref string) (*db.Article, error)
	UserByUsername(name string) (*db.User, error)
	UserByWikidotName(name string) (*db.User, error)
	SiteRatingMode() (string, error)
	CategoryRatingMode(category string) (string, error)
	VoteStats(articleID int64) (db.VoteStats, error)
	HiddenCategories(user *db.User) ([]string, error)
	ListArticles(f db.ListFilter, offset int, limit *int) ([]db.Article, error)
	CountArticles(f db.ListFilter, offset int, limit *int) (int, error)
}

type Query struct {
	Invalid bool

	Only        *db.Article
	HasOnly     bool
	FullName    string
	HasFullName bool

	Filter  db.ListFilter
	HasSort bool
	Offset  int
	Limit   *int
	Page    int
	PerPage int
}

const defaultPerPage = 20

type parser struct {
	src     Source
	article *db.Article
	viewer  *db.User
	params  map[string]string
	path    page.PathParams
	out     Query
	err     error
}

func Parse(src Source, article *db.Article, viewer *db.User, params map[string]string, pathParams page.PathParams) (Query, error) {
	p := &parser{src: src, article: article, viewer: viewer, params: params, path: pathParams}
	if p.params == nil {
		p.params = map[string]string{}
	}
	p.out.Page = 1
	p.out.PerPage = defaultPerPage

	if p.parseSinglePage() {
		return p.out, p.err
	}
	p.parseType()
	p.parseName()
	p.parseTags()
	p.parseCategory()
	p.parseParent()
	p.parseCreatedBy()
	p.parseCreatedAt()
	p.parseRating()
	p.parseVotes()
	p.parsePopularity()
	p.parseSort()
	p.parseWindow()
	if err := p.resolveRatingMode(); err != nil {
		return p.out, err
	}
	return p.out, p.err
}

func (p *parser) get(key string) string { return p.params[key] }

func (p *parser) invalid() { p.out.Invalid = true }

func (p *parser) fail(err error) {
	if p.err == nil {
		p.err = err
	}
}

func (p *parser) parseSinglePage() bool {
	if p.get("name") == "." || p.get("range") == "." || p.get("fullname") == "." {
		if p.article != nil {
			p.out.Only, p.out.HasOnly = p.article, true
		} else {
			p.invalid()
		}
		return true
	}
	if full := p.get("fullname"); full != "" {
		p.out.FullName, p.out.HasFullName = full, true
		return true
	}
	return false
}

func (p *parser) parseType() {
	pageType := p.get("pagetype")
	if _, ok := p.params["pagetype"]; !ok {
		pageType = db.PageTypeNormal
	}
	if pageType == db.PageTypeNormal || pageType == db.PageTypeHidden {
		p.out.Filter.PageType = pageType
	}
}

func (p *parser) parseName() {
	name, ok := p.params["name"]
	if !ok {
		return
	}
	if name == "*" {
		return
	}
	name = strings.ToLower(strings.ReplaceAll(name, "%", "*"))
	switch {
	case name == "=":
		if p.article == nil {
			p.invalid()
			return
		}
		p.out.Filter.Name, p.out.Filter.HasName = p.article.Name, true
	case strings.Contains(name, "*"):
		p.out.Filter.NamePrefix = name[:strings.Index(name, "*")]
		p.out.Filter.HasNamePrefix = true
	default:
		p.out.Filter.Name, p.out.Filter.HasName = name, true
	}
}

func (p *parser) parseTags() {
	raw, ok := p.params["tags"]
	if !ok || raw == "*" {
		return
	}
	raw = strings.ToLower(strings.ReplaceAll(raw, ",", " "))
	switch raw {
	case "-":
		p.out.Filter.NoTags = true
		return
	case "=", "==":
		if p.article == nil {
			p.invalid()
			return
		}
		ids, err := p.src.ArticleTagIDs(p.article.ID)
		if err != nil {
			p.fail(err)
			return
		}
		if raw == "=" {
			p.out.Filter.RequiredTags = ids
		} else {
			p.out.Filter.ExactTags = ids
		}
		return
	}

	var (
		requiredMissing bool
		presentNamed    int
		presentFound    int
	)
	for _, tag := range splitFields(raw) {
		switch {
		case strings.HasPrefix(tag, "-"):
			ids, err := p.tagIDs(tag[1:])
			if err != nil {
				p.fail(err)
				return
			}
			p.out.Filter.AbsentTags = append(p.out.Filter.AbsentTags, ids...)
		case strings.HasPrefix(tag, "+"):
			ids, err := p.tagIDs(tag[1:])
			if err != nil {
				p.fail(err)
				return
			}
			if len(ids) == 0 {
				requiredMissing = true
			}
			p.out.Filter.RequiredTags = append(p.out.Filter.RequiredTags, ids...)
		default:
			ids, err := p.tagIDs(tag)
			if err != nil {
				p.fail(err)
				return
			}
			presentNamed++
			presentFound += len(ids)
			p.out.Filter.PresentTags = append(p.out.Filter.PresentTags, ids...)
		}
	}
	// A tag nobody has ever used empties the listing only when it was required
	// or when it was the only thing asked for.
	if requiredMissing || (presentNamed > 0 && presentFound == 0) {
		p.invalid()
	}
}

func (p *parser) tagIDs(name string) ([]int64, error) {
	if strings.Contains(name, ":") {
		category, bare := splitName(name)
		return p.src.TagIDsByName(category, bare)
	}
	return p.src.TagIDsByName("", name)
}

func splitName(fullName string) (category, name string) {
	if before, after, found := strings.Cut(fullName, ":"); found {
		return before, after
	}
	return db.DefaultCategory, fullName
}

// The default is the page's own category, which is why a bare
// [[module ListPages]] lists siblings rather than the whole site.
func (p *parser) parseCategory() {
	raw, ok := p.params["category"]
	if !ok {
		raw = "."
	}
	if raw == "*" {
		return
	}
	raw = strings.ToLower(strings.ReplaceAll(raw, ",", " "))
	if raw == "." {
		if p.article == nil {
			p.invalid()
			return
		}
		p.out.Filter.Categories = []string{p.article.Category}
		return
	}
	for _, token := range strings.Split(raw, " ") {
		token, _, _ = strings.Cut(token, ":")
		if token == "" {
			continue
		}
		if token == "." {
			if p.article == nil {
				continue
			}
			token = p.article.Category
		}
		if strings.HasPrefix(token, "-") {
			p.out.Filter.NotCategories = append(p.out.Filter.NotCategories, token[1:])
		} else {
			p.out.Filter.Categories = append(p.out.Filter.Categories, token)
		}
	}
}

func (p *parser) parseParent() {
	raw := p.get("parent")
	if raw == "" {
		return
	}
	f := &p.out.Filter
	switch raw {
	case "-":
		f.HasParent = true
	case "=":
		f.HasParent, f.ParentID = true, p.articleParentID()
	case "-=":
		f.HasNotParent, f.NotParentID = true, p.articleParentID()
	case ".":
		if p.article == nil {
			p.invalid()
			return
		}
		f.HasParent, f.ParentID = true, &p.article.ID
	default:
		parent, err := p.src.ArticleByRef(strings.ToLower(raw))
		if errors.Is(err, db.ErrNotFound) {
			p.invalid()
			return
		}
		if err != nil {
			p.fail(err)
			return
		}
		f.HasParent, f.ParentID = true, &parent.ID
	}
}

func (p *parser) articleParentID() *int64 {
	if p.article == nil {
		return nil
	}
	return p.article.ParentID
}

func (p *parser) parseCreatedBy() {
	raw := p.get("created_by")
	if raw == "" {
		return
	}
	var (
		user *db.User
		err  error
	)
	if raw == "." {
		user = p.viewer
	} else {
		raw = strings.TrimSpace(raw)
		if wd, ok := strings.CutPrefix(raw, "wd:"); ok {
			user, err = p.src.UserByWikidotName(wd)
		} else {
			user, err = p.src.UserByUsername(raw)
		}
		if errors.Is(err, db.ErrNotFound) {
			user, err = nil, nil
		}
		if err != nil {
			p.fail(err)
			return
		}
	}
	if user == nil {
		p.invalid()
		return
	}
	p.out.Filter.AuthorID = &user.ID
}

func (p *parser) parseCreatedAt() {
	raw := p.get("created_at")
	if raw == "" {
		return
	}
	if strings.TrimSpace(raw) == "=" {
		if p.article == nil {
			p.invalid()
			return
		}
		y, m, d := p.article.CreatedAt.UTC().Date()
		dayStart := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		dayEnd := time.Date(y, m, d, 23, 59, 59, 0, time.UTC)
		p.out.Filter.CreatedAt = &db.TimeFilter{Op: db.TimeRange, Start: dayStart, End: dayEnd}
		return
	}

	op, rest := splitArgOperator(raw, []string{">=", "<=", "<>", ">", "<", "="}, "=")
	first, last, ok := parseDateBounds(strings.TrimSpace(rest))
	if !ok {
		p.invalid()
		return
	}
	p.out.Filter.CreatedAt = &db.TimeFilter{Op: timeOp(op), Start: first, End: last}
}

func timeOp(op string) string {
	switch op {
	case "<>":
		return db.TimeExcludeRange
	case "<":
		return db.TimeLT
	case ">":
		return db.TimeGT
	case "<=":
		return db.TimeLTE
	case ">=":
		return db.TimeGTE
	}
	return db.TimeRange
}

// Month and day are clamped rather than rejected, so 2020-13-99 is the last
// day of 2020-12.
func parseDateBounds(text string) (first, last time.Time, ok bool) {
	parts := strings.Split(text, "-")
	year, err := pyInt(parts[0])
	if err != nil || year < 1 || year > 9999 {
		return time.Time{}, time.Time{}, false
	}
	month, day := 1, 1
	lastMonth, lastDay := 12, 31

	if len(parts) >= 2 {
		m, err := pyInt(parts[1])
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		month = clamp(m, 1, 12)
		lastMonth = month
		lastDay = daysIn(year, month)
	}
	if len(parts) >= 3 {
		d, err := pyInt(parts[2])
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		day = clamp(d, 1, daysIn(year, month))
		lastDay = day
	}
	first = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	last = time.Date(year, time.Month(lastMonth), lastDay, 0, 0, 0, 0, time.UTC)
	return first, last, true
}

func daysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (p *parser) parseRating() {
	raw := p.get("rating")
	if raw == "" {
		return
	}
	if strings.TrimSpace(raw) == "=" {
		rating, ok := p.currentRating()
		if !ok {
			return
		}
		p.out.Filter.Rating = &db.NumFilter{Op: db.NumEQ, Value: ratingValue(rating)}
		return
	}
	op, rest := splitArgOperator(raw, []string{">=", "<=", "<>", ">", "<", "="}, "=")
	value, err := pyFloat(strings.TrimSpace(rest))
	if err != nil {
		p.invalid()
		return
	}
	p.out.Filter.Rating = &db.NumFilter{Op: numOp(op), Value: value}
}

func (p *parser) parseVotes() {
	raw := p.get("votes")
	if raw == "" {
		return
	}
	if strings.TrimSpace(raw) == "=" {
		rating, ok := p.currentRating()
		if !ok {
			return
		}
		p.out.Filter.Votes = &db.NumFilter{Op: db.NumEQ, Value: float64(rating.Votes)}
		return
	}
	op, rest := splitArgOperator(raw, []string{">=", "<=", "<>", ">", "<", "="}, "=")
	value, err := pyInt(strings.TrimSpace(rest))
	if err != nil {
		p.invalid()
		return
	}
	p.out.Filter.Votes = &db.NumFilter{Op: numOp(op), Value: float64(value)}
}

func (p *parser) parsePopularity() {
	raw := p.get("popularity")
	if raw == "" {
		return
	}
	if strings.TrimSpace(raw) == "=" {
		rating, ok := p.currentRating()
		if !ok {
			return
		}
		p.out.Filter.Popularity = &db.NumFilter{Op: db.NumEQ, Value: float64(rating.Popularity)}
		return
	}
	op, rest := splitArgOperator(raw, []string{">=", "<=", "<>", ">", "<", "="}, "=")
	value, err := pyInt(strings.TrimSpace(rest))
	if err != nil {
		p.invalid()
		return
	}
	p.out.Filter.Popularity = &db.NumFilter{Op: numOp(op), Value: float64(value)}
}

func (p *parser) currentRating() (page.Rating, bool) {
	if p.article == nil {
		p.invalid()
		return page.Rating{}, false
	}
	mode, err := p.ratingModeOf(p.article.Category)
	if err != nil {
		p.fail(err)
		return page.Rating{}, false
	}
	if mode == page.RatingModeDisabled {
		return page.DisabledRating(), true
	}
	stats, err := p.src.VoteStats(p.article.ID)
	if err != nil {
		p.fail(err)
		return page.Rating{}, false
	}
	return page.RatingOf(mode, stats), true
}

func ratingValue(r page.Rating) float64 {
	switch value := r.Value.(type) {
	case int:
		return float64(value)
	case float64:
		return value
	}
	return 0
}

// Python splits on a single space and strips each piece, which keeps a tab
// inside a tag name rather than breaking on it.
func splitFields(s string) []string {
	var out []string
	for _, part := range strings.Split(s, " ") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func numOp(op string) string {
	switch op {
	case "<>":
		return db.NumNE
	case "<":
		return db.NumLT
	case ">":
		return db.NumGT
	case "<=":
		return db.NumLTE
	case ">=":
		return db.NumGTE
	}
	return db.NumEQ
}

func (p *parser) parseSort() {
	raw, ok := p.params["order"]
	if !ok {
		raw = "created_at desc"
	}
	fields := strings.Split(raw, " ")
	ascending := true
	if len(fields) == 2 && fields[1] == "desc" {
		ascending = false
	}
	p.out.Filter.Sort = db.Sort{Column: fields[0], Ascending: ascending}
	p.out.HasSort = true
}

func (p *parser) parseWindow() {
	if offset, err := pyInt(p.getOr("offset", "0")); err == nil {
		p.out.Offset = offset
	}
	if raw, ok := p.params["limit"]; ok {
		if limit, err := pyInt(raw); err == nil {
			p.out.Limit = &limit
		}
	}
	perPage, err := pyInt(p.getOr("perpage", "20"))
	if err != nil {
		perPage = defaultPerPage
	}
	pageNum, err := pyInt(p.pathOr("p", "1"))
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	p.out.Page = pageNum
	p.out.PerPage = perPage
}

func (p *parser) getOr(key, def string) string {
	if value, ok := p.params[key]; ok {
		return value
	}
	return def
}

func (p *parser) pathOr(key, def string) string {
	if param, ok := p.path.Lookup(key); ok {
		return param.Value
	}
	return def
}

// The first named category decides, which is how one listing can rate its
// pages differently from the page it sits on.
func (p *parser) resolveRatingMode() error {
	f := &p.out.Filter
	needs := f.Rating != nil || f.Popularity != nil ||
		f.Sort.Column == db.SortRating || f.Sort.Column == db.SortPopularity
	if !needs {
		return nil
	}
	category := db.DefaultCategory
	if len(f.Categories) > 0 && f.Categories[0] != "" {
		category = f.Categories[0]
	}
	mode, err := p.ratingModeOf(category)
	if err != nil {
		return err
	}
	f.RatingMode = mode
	return nil
}

func (p *parser) ratingModeOf(category string) (string, error) {
	site, err := p.src.SiteRatingMode()
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	own, err := p.src.CategoryRatingMode(category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	return page.RatingMode(site, own), nil
}

// The prefixes are tried in the order given, so the two-character ones have
// to come first.
func splitArgOperator(arg string, allowed []string, def string) (op, rest string) {
	for _, candidate := range allowed {
		if strings.HasPrefix(arg, candidate) {
			return candidate, arg[len(candidate):]
		}
	}
	return def, arg
}

var errNotANumber = errors.New("listpages: not a number")

// Python's int() tolerates surrounding space, a sign and underscores between
// digits. Non-ASCII digits are the one thing it takes that this does not.
func pyInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	body := strings.TrimPrefix(strings.TrimPrefix(s, "+"), "-")
	if body == "" || strings.HasPrefix(body, "_") || strings.HasSuffix(body, "_") {
		return 0, errNotANumber
	}
	if strings.Contains(body, "__") {
		return 0, errNotANumber
	}
	for _, r := range body {
		if r != '_' && (r < '0' || r > '9') {
			return 0, errNotANumber
		}
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, "_", ""))
	if err != nil {
		return 0, errNotANumber
	}
	return n, nil
}

// Go would take a hexadecimal float that Python rejects, so the digits are
// checked first.
func pyFloat(s string) (float64, error) {
	if n, err := pyInt(s); err == nil {
		return float64(n), nil
	}
	s = strings.TrimSpace(s)
	lowered := strings.ToLower(s)
	if strings.Contains(lowered, "x") || strings.Contains(lowered, "p") {
		return 0, errNotANumber
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, "_", ""), 64)
	if err != nil {
		return 0, errNotANumber
	}
	return f, nil
}
