// Package shell renders the HTML page that a rendered article is placed into.
package shell

import (
	"embed"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

const (
	licenseURL  = "http://creativecommons.org/licenses/by-sa/3.0/"
	licenseName = "Creative Commons Attribution-ShareAlike 3.0 License"
)

//go:embed templates/*.html
var files embed.FS

// text/template rather than html/template because Go's escaper picks and spells
// different characters than these pages do, and the output is compared byte for
// byte.
var tmpl = template.Must(template.New("shell").Funcs(template.FuncMap{
	"esc":   escape.HTML,
	"escjs": escape.JS,
	"sub":   func(a, b int) int { return a - b },
	"feed":  newFeed,
}).ParseFS(files, "templates/*.html"))

type Breadcrumb struct {
	URL   string
	Title string
}

type Tag struct {
	Name     string
	FullName string
}

// Hidden marks the tags Wikidot keeps out of the tag block.
func (t Tag) Hidden() bool { return strings.HasPrefix(t.Name, "_") }

type TagCategory struct {
	Name string
	Tags []Tag
}

// Data is one page's worth of shell. NavTop, NavSide and Content arrive as
// rendered HTML and are written through untouched.
type Data struct {
	SiteName     string
	SiteHeadline string
	SiteTitle    string
	SiteIcon     string

	// An empty one falls back to the built-in line.
	License string

	OGTitle       string
	OGDescription string
	OGImage       string
	OGURL         string

	NoIndex     bool
	GoogleTagID string

	ThemeURL      string
	ComputedStyle string

	NavTop  string
	NavSide string

	Title         string
	Content       string
	Breadcrumbs   []Breadcrumb
	TagCategories []TagCategory

	RevNumber int
	UpdatedAt time.Time

	LoginStatusConfig string
	OptionsConfig     string
}

type NotFound struct {
	PageID      string
	AllowCreate bool
	Options     string
}

type Renderer struct {
	loc    *i18n.Localizer
	assets *static.Assets
	tz     *time.Location
}

func New(loc *i18n.Localizer, assets *static.Assets, tz *time.Location) *Renderer {
	return &Renderer{loc: loc, assets: assets, tz: tz}
}

func (r *Renderer) Page(w io.Writer, d Data) error {
	return tmpl.ExecuteTemplate(w, "page.html", view{Data: d, r: r})
}

func (r *Renderer) NotFound(d NotFound) (string, error) {
	return r.execute("page_404.html", struct {
		AllowCreate bool
		Options     string
		Message     string
	}{
		AllowCreate: d.AllowCreate,
		Options:     d.Options,
		Message:     r.loc.T("page.not-found", "page", em(d.PageID)),
	})
}

func (r *Renderer) Forbidden(pageID string) (string, error) {
	return r.execute("page_403.html", struct{ Message string }{
		Message: r.loc.T("page.forbidden", "page", em(pageID)),
	})
}

func (r *Renderer) execute(name string, data any) (string, error) {
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func em(s string) string { return "<em>" + escape.HTML(s) + "</em>" }

type view struct {
	Data
	r *Renderer
}

func (v view) Lang() string { return v.r.loc.Lang() }

func (v view) T(id string, args ...any) string { return v.r.loc.T(id, args...) }

func (v view) Asset(name string) string { return v.r.assets.URL(name) }

func (v view) LicenseHTML() string {
	if v.Data.License != "" {
		return v.Data.License
	}
	return v.T("page.license", "link",
		`<a href="`+licenseURL+`" target="_blank">`+licenseName+`</a>`)
}

func (v view) PageInfo() string {
	date := `<span class="odate w-date" data-timestamp="` +
		strconv.FormatInt(v.UpdatedAt.Unix(), 10) + `000" data-format="` +
		escape.HTML(v.T("page.date-format-js")) + `" style="display: inline">` +
		escape.HTML(v.date()) + `</span>`
	return v.T("page.page-info", "rev", v.RevNumber, "date", date)
}

func (v view) date() string {
	at := v.UpdatedAt.In(v.r.tz)
	return v.T("page.date-format",
		"year", strconv.Itoa(at.Year()),
		"month", strconv.Itoa(int(at.Month())),
		"day", strconv.Itoa(at.Day()),
		"hour", fmt.Sprintf("%02d", at.Hour()),
		"minute", fmt.Sprintf("%02d", at.Minute()))
}

// System is the page the wiki's own pages sit in, which is not the one an
// article sits in.
type System struct {
	Title     string
	ThemeURL  string
	Before    string
	Heading   string
	BodyClass string
	BackLink  bool
	Content   string
}

type Reactive struct {
	ThemeURL string
	Config   string
}

type Profile struct {
	ID          int64
	DisplayName string
	Avatar      string
	Subtitle    string
	AuthIcon    string

	IsSelf           bool
	CanDirectMessage bool
	IsBlocked        bool
	ActionsConfig    string

	FromWikidot bool
	IsBot       bool
	FullName    string
	Bio         string
	BioHTML     string
	JoinedAt    time.Time

	Roles []ProfileRoles
	Edits ProfileFeed
	Posts ProfileFeed
}

type ProfileRoles struct {
	Site  string
	URL   string
	Names string
}

type ProfileFeed struct {
	Items      []ProfileItem
	Pagination string
}

type ProfileFlag struct {
	ID   string
	Desc string
}

type ProfileItem struct {
	URL     string
	Title   string
	Site    string
	At      time.Time
	Flags   []ProfileFlag
	Comment string
}

type feed struct {
	View  profileView
	Label string
	Feed  ProfileFeed
	Empty string
}

func newFeed(v profileView, label string, f ProfileFeed, empty string) feed {
	return feed{View: v, Label: label, Feed: f, Empty: empty}
}

type Login struct {
	AuthIcon  string
	SiteTitle string

	Username string
	CSRF     string
	Error    string
}

type Signup struct {
	AuthIcon  string
	SiteTitle string
	Notice    string

	Username  string
	IsWikidot bool
	CSRF      string
	Error     string
}

type Reset struct {
	Stage string
	CSRF  string
	Error string
}

type Accept struct {
	Username  string
	IsWikidot bool
	CSRF      string
	Error     string
	Fatal     bool
}

type ProfileEdit struct {
	AuthIcon    string
	DisplayName string
	Avatar      string
	ProfileURL  string

	FullName string
	Bio      string

	AdvancedEditor bool

	CSRF  string
	Error string
	Saved bool
}

func (r *Renderer) SystemPage(w io.Writer, d System) error {
	return tmpl.ExecuteTemplate(w, "system.html", systemView{System: d, r: r})
}

func (r *Renderer) Reactive(w io.Writer, d Reactive) error {
	return tmpl.ExecuteTemplate(w, "reactive.html", reactiveView{Reactive: d, r: r})
}

func (r *Renderer) Profile(d Profile) (string, error) {
	return r.execute("profile.html", profileView{Profile: d, r: r})
}

func (r *Renderer) ProfileEdit(d ProfileEdit) (string, error) {
	return r.execute("profile_edit.html", profileEditView{ProfileEdit: d, r: r})
}

func (r *Renderer) Login(d Login) (string, error) {
	return r.execute("login.html", loginView{Login: d, r: r})
}

func (r *Renderer) Signup(d Signup) (string, error) {
	return r.execute("signup.html", signupView{Signup: d, r: r})
}

func (r *Renderer) Accept(d Accept) (string, error) {
	return r.execute("accept.html", acceptView{Accept: d, r: r})
}

func (r *Renderer) Reset(d Reset) (string, error) {
	return r.execute("reset.html", resetView{Reset: d, r: r})
}

type resetView struct {
	Reset
	r *Renderer
}

func (v resetView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v resetView) Asset(name string) string        { return v.r.assets.URL(name) }

type signupView struct {
	Signup
	r *Renderer
}

func (v signupView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v signupView) Asset(name string) string        { return v.r.assets.URL(name) }

type acceptView struct {
	Accept
	r *Renderer
}

func (v acceptView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v acceptView) Asset(name string) string        { return v.r.assets.URL(name) }

type loginView struct {
	Login
	r *Renderer
}

func (v loginView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v loginView) Asset(name string) string        { return v.r.assets.URL(name) }

type profileEditView struct {
	ProfileEdit
	r *Renderer
}

func (v profileEditView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v profileEditView) Asset(name string) string        { return v.r.assets.URL(name) }
func (v profileEditView) Initial() string                 { return initial(v.DisplayName) }

type systemView struct {
	System
	r *Renderer
}

func (v systemView) Lang() string                    { return v.r.loc.Lang() }
func (v systemView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v systemView) Asset(name string) string        { return v.r.assets.URL(name) }

type reactiveView struct {
	Reactive
	r *Renderer
}

func (v reactiveView) Lang() string             { return v.r.loc.Lang() }
func (v reactiveView) Asset(name string) string { return v.r.assets.URL(name) }

type profileView struct {
	Profile
	r *Renderer
}

func (v profileView) T(id string, args ...any) string { return v.r.loc.T(id, args...) }
func (v profileView) Asset(name string) string        { return v.r.assets.URL(name) }

func (v profileView) Initial() string { return initial(v.DisplayName) }

// The letter the avatar box falls back to is the display name's first
// character rather than its first byte.
func initial(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return ""
}

func (v profileView) ODate(at time.Time) string {
	return `<span class="odate w-date" data-timestamp="` +
		strconv.FormatInt(at.Unix(), 10) + `000" data-format="` +
		escape.HTML(v.T("profile.date-format-js")) + `" style="display: inline">` +
		escape.HTML(v.date(at)) + `</span>`
}

func (v profileView) date(at time.Time) string {
	at = at.In(v.r.tz)
	return v.T("profile.date-format",
		"year", strconv.Itoa(at.Year()),
		"month", strconv.Itoa(int(at.Month())),
		"day", strconv.Itoa(at.Day()))
}
