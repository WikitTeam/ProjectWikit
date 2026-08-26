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
// different characters than Django's, and this output is compared byte for byte.
var tmpl = template.Must(template.New("shell").Funcs(template.FuncMap{
	"esc":   escape.HTML,
	"escjs": escape.JS,
	"sub":   func(a, b int) int { return a - b },
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

func (v view) License() string {
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
