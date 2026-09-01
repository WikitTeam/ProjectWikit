package page

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
)

const (
	RatingModeDefault  = "default"
	RatingModeDisabled = "disabled"
	RatingModeUpDown   = "updown"
	RatingModeStars    = "stars"
)

// VarSource is everything a page variable may have to read. Each method is one
// query, because a page that mentions no variable must issue none of them.
type VarSource interface {
	LatestSource(articleID int64) (string, error)
	Authors(articleID int64) ([]db.User, error)
	LatestEditor(articleID int64) (*db.User, error)
	RevisionCount(articleID int64) (int, error)
	Tags(articleID int64) ([]string, error)
	VoteStats(articleID int64) (db.VoteStats, error)
	SiteRatingMode() (string, error)
	CategoryRatingMode(category string) (string, error)
	HasVoted(articleID int64, userID *int64) (bool, error)
	ArticleByID(id int64) (*db.Article, error)
	CategoryForm(category string) (*form.Definition, error)
	SiteName() string
}

// Vars resolves the page variables of one article. Every value is computed at
// most once and only when asked for, because content pulls a whole revision and
// the rating runs an aggregate.
type Vars struct {
	article *db.Article
	user    *db.User
	src     VarSource
	loc     *i18n.Localizer

	values map[string]string
	err    error

	source      string
	sourceDone  bool
	formDef     *form.Definition
	formDefDone bool
	formValues  map[string]string
	authors     []db.User
	authorsDone bool
	editor      *db.User
	editorDone  bool
	votes       db.VoteStats
	votesDone   bool
	parent      *db.Article
	parentDone  bool
}

func NewVars(article *db.Article, user *db.User, src VarSource, loc *i18n.Localizer) *Vars {
	return &Vars{article: article, user: user, src: src, loc: loc, values: map[string]string{}}
}

// Err reports the first query that failed. A failed lookup leaves the variable
// unresolved, so the page still renders, with %%name%% in it, unless the caller
// checks this.
func (v *Vars) Err() error { return v.err }

func (v *Vars) fail(err error) bool {
	if v.err == nil {
		v.err = err
	}
	return false
}

func (v *Vars) text(id string) string {
	if v.loc == nil {
		return id
	}
	return v.loc.T(id)
}

// Lookup resolves a bare %%name%%. The name is matched exactly, so %%Title%% is
// not %%title%%.
func (v *Vars) Lookup(name string) (string, bool) {
	if v == nil || v.article == nil {
		return "", false
	}
	if value, ok := v.values[name]; ok {
		return value, true
	}
	if value, ok := v.compute(name); ok {
		v.values[name] = value
		return value, true
	}
	if format, ok := strings.CutPrefix(name, "created_at|"); ok {
		return dateWithFormat(v.article.CreatedAt.Unix(), format), true
	}
	if format, ok := strings.CutPrefix(name, "updated_at|"); ok {
		return dateWithFormat(v.article.UpdatedAt.Unix(), format), true
	}
	return "", false
}

// This resolves %%this|name%%, the form an included page uses to reach the page
// that included it. Unlike a bare name this one is case insensitive and takes
// no date format.
func (v *Vars) This(param string) (string, bool) {
	name, ok := strings.CutPrefix(param, "this|")
	if !ok {
		return "", false
	}
	if v == nil || v.article == nil {
		return "", false
	}
	name = strings.ToLower(name)
	if value, ok := v.values[name]; ok {
		return value, true
	}
	value, ok := v.compute(name)
	if ok {
		v.values[name] = value
	}
	return value, ok
}

func dateWithFormat(timestamp int64, format string) string {
	return fmt.Sprintf(`[[date %d format="%s"]]`, timestamp, strings.TrimSpace(format))
}

func (v *Vars) compute(name string) (string, bool) {
	a := v.article
	switch name {
	case "name":
		return a.Name, true
	case "category":
		return a.Category, true
	case "fullname":
		return a.FullName(), true
	case "title":
		return a.Title, true
	case "title_linked", "linked_title":
		return "[[[" + a.FullName() + "|]]]", true
	case "link":
		// Wikidot builds this from the page name; here it is the title, so it
		// only lines up on pages whose title was never edited.
		return "/" + a.Title, true
	case "site_name":
		return v.src.SiteName(), true
	case "content":
		return v.latestSource()
	case "rating":
		return v.formattedRating()
	case "rating_votes":
		votes, ok := v.voteStats()
		if !ok {
			return "", false
		}
		return strconv.Itoa(votes.Count), true
	case "popularity":
		return v.popularity()
	case "current_user_voted":
		var userID *int64
		if v.user != nil {
			userID = &v.user.ID
		}
		voted, err := v.src.HasVoted(a.ID, userID)
		if err != nil {
			return "", v.fail(err)
		}
		if voted {
			return "True", true
		}
		return "False", true
	case "revisions":
		n, err := v.src.RevisionCount(a.ID)
		if err != nil {
			return "", v.fail(err)
		}
		return strconv.Itoa(n), true
	case "created_by":
		authors, ok := v.getAuthors()
		if !ok {
			return "", false
		}
		parts := make([]string, len(authors))
		for i := range authors {
			parts[i] = v.userText(&authors[i])
		}
		return strings.Join(parts, " "), true
	case "created_by_linked":
		return v.authorTags(false)
	case "created_by_linked_plain":
		return v.authorTags(true)
	case "updated_by":
		editor, ok := v.getEditor()
		if !ok {
			return "", false
		}
		return v.userText(editor), true
	case "updated_by_linked":
		return v.editorTag(false)
	case "updated_by_linked_plain":
		return v.editorTag(true)
	case "authors_count":
		authors, ok := v.getAuthors()
		if !ok {
			return "", false
		}
		return strconv.Itoa(len(authors)), true
	case "tags":
		tags, ok := v.getTags()
		if !ok {
			return "", false
		}
		return strings.Join(tags, " "), true
	case "tags_linked":
		tags, ok := v.getTags()
		if !ok {
			return "", false
		}
		parts := make([]string, len(tags))
		for i, tag := range tags {
			parts[i] = "[/system:page-tags/tag/" + QuoteAll(tag) + " " + tag + "]"
		}
		return strings.Join(parts, ", "), true
	case "created_at":
		return fmt.Sprintf("[[date %d]]", a.CreatedAt.Unix()), true
	case "updated_at":
		return fmt.Sprintf("[[date %d]]", a.UpdatedAt.Unix()), true
	case "parent_name", "parent_category", "parent_fullname", "parent_title",
		"parent_title_linked", "parent_linked_title":
		return v.parentVar(name)
	}
	if n, ok := sectionIndex(name); ok {
		return v.contentSection(n)
	}
	if kind, field, ok := form.ParseVar(name); ok {
		return v.formVar(kind, field)
	}
	return "", false
}

func (v *Vars) formVar(kind, field string) (string, bool) {
	def, ok := v.definition()
	if !ok || def == nil {
		return "", false
	}
	switch kind {
	case form.VarLabel:
		return def.Label(field)
	case form.VarHint:
		return def.Hint(field)
	}
	values, ok := v.storedValues()
	if !ok {
		return "", false
	}
	if kind == form.VarRaw {
		return def.Raw(values, field)
	}
	return def.Data(values, field)
}

func (v *Vars) definition() (*form.Definition, bool) {
	if v.formDefDone {
		return v.formDef, true
	}
	def, err := v.src.CategoryForm(v.article.Category)
	if err != nil {
		return nil, v.fail(err)
	}
	v.formDef, v.formDefDone = def, true
	return def, true
}

// A page of a form category stores its fields where an ordinary page stores
// wikitext, so the values come out of the same revision.
func (v *Vars) storedValues() (map[string]string, bool) {
	if v.formValues != nil {
		return v.formValues, true
	}
	source, ok := v.latestSource()
	if !ok {
		// A page with no revision yet still answers with the defaults.
		source = ""
	}
	values, err := form.ParseData(source)
	if err != nil {
		values = map[string]string{}
	}
	v.formValues = values
	return values, true
}

// Four is the marker authors write, and a longer run is taken too because a
// line of equal signs means nothing else in wikitext.
var sectionBreak = regexp.MustCompile(`(?m)^={4,}[ \t]*\r?$`)

func sectionIndex(name string) (int, bool) {
	arg, ok := strings.CutPrefix(name, "content{")
	if !ok {
		return 0, false
	}
	arg, ok = strings.CutSuffix(arg, "}")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func (v *Vars) contentSection(n int) (string, bool) {
	source, ok := v.latestSource()
	if !ok {
		return "", false
	}
	sections := sectionBreak.Split(source, -1)
	// An index past the last section resolves to empty rather than staying
	// unresolved, so a section nobody has written yet renders blank.
	if n > len(sections) {
		return "", true
	}
	return strings.Trim(sections[n-1], "\r\n"), true
}

func (v *Vars) latestSource() (string, bool) {
	if v.sourceDone {
		return v.source, true
	}
	source, err := v.src.LatestSource(v.article.ID)
	if err != nil {
		return "", v.notFoundOrFail(err)
	}
	v.source, v.sourceDone = source, true
	return source, true
}

// A missing row is kept out of Err, so a page with no revision yet leaves
// %%content%% standing.
func (v *Vars) notFoundOrFail(err error) bool {
	if errors.Is(err, db.ErrNotFound) {
		return false
	}
	return v.fail(err)
}

func (v *Vars) parentVar(name string) (string, bool) {
	if v.article.ParentID == nil {
		return "", false
	}
	if !v.parentDone {
		parent, err := v.src.ArticleByID(*v.article.ParentID)
		if err != nil {
			return "", v.notFoundOrFail(err)
		}
		v.parent, v.parentDone = parent, true
	}
	p := v.parent
	switch name {
	case "parent_name":
		return p.Name, true
	case "parent_category":
		return p.Category, true
	case "parent_fullname":
		return p.FullName(), true
	case "parent_title":
		return p.Title, true
	case "parent_title_linked", "parent_linked_title":
		return "[[[" + p.FullName() + "|" + p.Title + "]]]", true
	}
	return "", false
}

func (v *Vars) getAuthors() ([]db.User, bool) {
	if v.authorsDone {
		return v.authors, true
	}
	authors, err := v.src.Authors(v.article.ID)
	if err != nil {
		return nil, v.fail(err)
	}
	v.authors, v.authorsDone = authors, true
	return authors, true
}

func (v *Vars) getEditor() (*db.User, bool) {
	if v.editorDone {
		return v.editor, true
	}
	editor, err := v.src.LatestEditor(v.article.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, v.fail(err)
	}
	v.editor, v.editorDone = editor, true
	return editor, true
}

func (v *Vars) getTags() ([]string, bool) {
	tags, err := v.src.Tags(v.article.ID)
	if err != nil {
		return nil, v.fail(err)
	}
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = strings.ToLower(tag)
	}
	sort.Strings(out)
	return out, true
}

func (v *Vars) voteStats() (db.VoteStats, bool) {
	if v.votesDone {
		return v.votes, true
	}
	votes, err := v.src.VoteStats(v.article.ID)
	if err != nil {
		return db.VoteStats{}, v.fail(err)
	}
	v.votes, v.votesDone = votes, true
	return votes, true
}

func (v *Vars) ratingMode() (string, bool) {
	site, err := v.src.SiteRatingMode()
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", v.fail(err)
	}
	category, err := v.src.CategoryRatingMode(v.article.Category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", v.fail(err)
	}
	return RatingMode(site, category), true
}

// RatingMode walks the settings chain, where the built-in default is updown and
// a level that says default hands the question to the level above it.
func RatingMode(siteMode, categoryMode string) string {
	mode := RatingModeUpDown
	if siteMode != "" && siteMode != RatingModeDefault {
		mode = siteMode
	}
	if categoryMode != "" && categoryMode != RatingModeDefault {
		mode = categoryMode
	}
	return mode
}

func (v *Vars) formattedRating() (string, bool) {
	mode, ok := v.ratingMode()
	if !ok {
		return "", false
	}
	if mode == RatingModeDisabled {
		return "0", true
	}
	votes, ok := v.voteStats()
	if !ok {
		return "", false
	}
	switch mode {
	case RatingModeUpDown:
		return fmt.Sprintf("%+d", int(votes.Sum)), true
	case RatingModeStars:
		if votes.Count == 0 {
			return "—", true
		}
		return fmt.Sprintf("%.1f", votes.Average), true
	}
	return "0", true
}

func (v *Vars) popularity() (string, bool) {
	mode, ok := v.ratingMode()
	if !ok {
		return "", false
	}
	if mode == RatingModeDisabled {
		return "0", true
	}
	votes, ok := v.voteStats()
	if !ok {
		return "", false
	}
	good := votes.GoodUpDown
	if mode == RatingModeStars {
		good = votes.GoodStars
	}
	count := votes.Count
	if count == 0 {
		count = 1
	}
	return strconv.Itoa(roundHalfEven(float64(good) / float64(count) * 100)), true
}

// A halfway value goes to the even neighbour, so a popularity of exactly 12.5
// is 12 and not 13.
func roundHalfEven(x float64) int {
	return int(math.RoundToEven(x))
}

func (v *Vars) authorTags(plain bool) (string, bool) {
	authors, ok := v.getAuthors()
	if !ok {
		return "", false
	}
	parts := make([]string, len(authors))
	for i := range authors {
		parts[i] = userTag(&authors[i], plain)
	}
	return strings.Join(parts, " "), true
}

func (v *Vars) editorTag(plain bool) (string, bool) {
	editor, ok := v.getEditor()
	if !ok {
		return "", false
	}
	if editor == nil {
		return v.userText(nil), true
	}
	return userTag(editor, plain), true
}

func userTag(u *db.User, plain bool) string {
	if plain {
		return "[[user " + u.Username + "]]"
	}
	return "[[*user " + u.Username + "]]"
}

func (v *Vars) userText(u *db.User) string {
	if u == nil {
		return v.text("user-system")
	}
	if u.Type == db.UserTypeWikidot {
		if u.DisplayName != "" {
			return "wd:" + u.DisplayName
		}
		return "wd:" + u.WikidotUsername
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}

func QuoteAll(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}
