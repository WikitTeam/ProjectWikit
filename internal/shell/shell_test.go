package shell

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

var update = flag.Bool("update", false, "rewrite the golden file and the corpus the oracle reads")

const (
	goldenPath = "testdata/shell.golden"
	corpusPath = "testdata/shell_corpus.json"
)

type crumbSpec struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type tagSpec struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type tagCategorySpec struct {
	Name string    `json:"name"`
	Tags []tagSpec `json:"tags"`
}

type pageSpec struct {
	SiteName          string            `json:"site_name"`
	SiteHeadline      string            `json:"site_headline"`
	SiteTitle         string            `json:"site_title"`
	SiteIcon          string            `json:"site_icon"`
	OGTitle           string            `json:"og_title"`
	OGDescription     string            `json:"og_description"`
	OGImage           string            `json:"og_image"`
	OGURL             string            `json:"og_url"`
	NoIndex           bool              `json:"noindex"`
	GoogleTagID       string            `json:"google_tag_id"`
	ThemeURL          string            `json:"theme_url"`
	ComputedStyle     string            `json:"computed_style"`
	NavTop            string            `json:"nav_top"`
	NavSide           string            `json:"nav_side"`
	Title             string            `json:"title"`
	Content           string            `json:"content"`
	Breadcrumbs       []crumbSpec       `json:"breadcrumbs"`
	TagCategories     []tagCategorySpec `json:"tags_categories"`
	RevNumber         int               `json:"rev_number"`
	UpdatedAt         string            `json:"updated_at"`
	LoginStatusConfig string            `json:"login_status_config"`
	OptionsConfig     string            `json:"options_config"`
}

type caseSpec struct {
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Page        *pageSpec `json:"page,omitempty"`
	PageID      string    `json:"page_id,omitempty"`
	AllowCreate bool      `json:"allow_create,omitempty"`
	Options     string    `json:"options,omitempty"`
}

type corpusFile struct {
	Assets map[string]string `json:"assets"`
	Cases  []caseSpec        `json:"cases"`
}

var bundle = fstest.MapFS{
	"app.js":                  {Data: []byte("console.log(1)\n")},
	"app.css":                 {Data: []byte("body{color:#000}\n")},
	"wikidot-base.css":        {Data: []byte("#page-content{margin:0}\n")},
	"fontawesome/css/all.css": {Data: []byte(".fa{font-family:fa}\n")},
}

func testRenderer(t *testing.T) *Renderer {
	t.Helper()
	b, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return New(b.Localizer(i18n.DefaultLanguage), static.NewAssets(bundle), time.FixedZone("Asia/Shanghai", 8*60*60))
}

func base() pageSpec {
	return pageSpec{
		SiteName:          "Wikit",
		SiteHeadline:      "a wiki",
		SiteTitle:         "Wikit",
		SiteIcon:          "-/sites/icon.png",
		OGTitle:           "Wikit",
		OGDescription:     "an excerpt",
		OGURL:             "https://wiki.example/main",
		ThemeURL:          "/-/static/theme.css",
		NavTop:            `<div class="top-bar"><ul><li><a href="/main">Main</a></li></ul></div>`,
		NavSide:           `<div class="side-block"><p>side</p></div>`,
		Content:           `<p>hello <em>world</em></p>`,
		LoginStatusConfig: `{"user": {"type": "anonymous"}, "notificationCount": 0}`,
		OptionsConfig:     `{"optionsEnabled": true, "pageId": "main"}`,
	}
}

func withPage(name string, edit func(p *pageSpec)) caseSpec {
	p := base()
	edit(&p)
	return caseSpec{Name: name, Kind: "page", Page: &p}
}

func corpus() corpusFile {
	assets := static.NewAssets(bundle)
	names := []string{"app.js", "app.css", "wikidot-base.css", "fontawesome/css/all.css"}
	urls := make(map[string]string, len(names))
	for _, name := range names {
		urls[name] = assets.URL(name)
	}

	crumbs := []crumbSpec{
		{URL: "/main", Title: "Main"},
		{URL: "/main:parent", Title: "Parent"},
		{URL: "/main:child", Title: "Child"},
	}
	tags := []tagCategorySpec{
		{Name: "默认", Tags: []tagSpec{{Name: "scp", FullName: "scp"}, {Name: "euclid", FullName: "euclid"}}},
		{Name: "attribute", Tags: []tagSpec{{Name: "co-authored", FullName: "attribute:co-authored"}, {Name: "_hidden", FullName: "attribute:_hidden"}}},
	}

	return corpusFile{
		Assets: urls,
		Cases: []caseSpec{
			withPage("minimal", func(p *pageSpec) {
				p.OGDescription = ""
				p.NavTop = ""
				p.NavSide = ""
				p.Content = ""
				p.LoginStatusConfig = "{}"
				p.OptionsConfig = "{}"
			}),
			withPage("full", func(p *pageSpec) {
				p.Title = "Main page"
				p.OGTitle = "Main page"
				p.OGImage = "https://wiki.example/local--files/main/cover.png"
				p.ComputedStyle = "#page-content .x { color: red }"
				p.Breadcrumbs = crumbs
				p.TagCategories = tags
				p.RevNumber = 12
				p.UpdatedAt = "2026-08-25T14:03:07+08:00"
			}),
			withPage("escaping", func(p *pageSpec) {
				p.SiteName = `Wik<it> & "co"`
				p.SiteHeadline = "it's a wiki"
				p.SiteTitle = `<script>alert(1)</script>`
				p.SiteIcon = `-/sites/it's.png`
				p.Title = `A & B <b>`
				p.OGTitle = `A & B <b>`
				p.OGDescription = `it's "quoted" & <marked>`
				p.OGURL = "https://wiki.example/main?a=1&b=2"
				p.ThemeURL = "/-/theme/dark.css?v=1&x=2"
				p.Breadcrumbs = []crumbSpec{
					{URL: "/a?x=1&y=2", Title: `it's <first>`},
					{URL: "/b", Title: `"last" & final`},
				}
				p.TagCategories = []tagCategorySpec{
					{Name: `cat & "co"`, Tags: []tagSpec{{Name: "a&b", FullName: "x:a&b"}}},
				}
				p.LoginStatusConfig = `{"user": {"name": "it's <me>"}}`
				p.OptionsConfig = `{"pageId": "a&b"}`
			}),
			withPage("noindex", func(p *pageSpec) { p.NoIndex = true }),
			withPage("google_tag", func(p *pageSpec) { p.GoogleTagID = "G-AB1'2-3" }),
			withPage("one_breadcrumb", func(p *pageSpec) { p.Breadcrumbs = crumbs[:1] }),
			withPage("two_breadcrumbs", func(p *pageSpec) { p.Breadcrumbs = crumbs[:2] }),
			withPage("hidden_tags_only", func(p *pageSpec) {
				p.TagCategories = []tagCategorySpec{
					{Name: "attribute", Tags: []tagSpec{{Name: "_hidden", FullName: "attribute:_hidden"}}},
				}
			}),
			withPage("unpadded_date", func(p *pageSpec) {
				p.RevNumber = 0
				p.UpdatedAt = "2026-01-05T09:07:00+08:00"
			}),
			withPage("date_crosses_midnight_in_utc", func(p *pageSpec) {
				p.RevNumber = 3
				p.UpdatedAt = "2026-03-01T00:30:00+08:00"
			}),
			withPage("unicode", func(p *pageSpec) {
				p.SiteName = "维基"
				p.SiteHeadline = "一个维基站点"
				p.SiteTitle = "首页 - 维基"
				p.Title = "首页"
				p.TagCategories = []tagCategorySpec{
					{Name: "分类", Tags: []tagSpec{{Name: "中文标签", FullName: "分类:中文标签"}}},
				}
			}),
			{Name: "not_found", Kind: "not_found", PageID: "no-such-page"},
			{Name: "not_found_escaped", Kind: "not_found", PageID: `a&b<c>'d'`},
			{
				Name:        "not_found_create",
				Kind:        "not_found",
				PageID:      "new-page",
				AllowCreate: true,
				Options:     `{"page_id": "new-page", "pathParams": {"a": "b"}}`,
			},
			{Name: "forbidden", Kind: "forbidden", PageID: "secret:page"},
			{Name: "forbidden_escaped", Kind: "forbidden", PageID: `a&b<c>'d'`},
		},
	}
}

func dataFor(t *testing.T, p *pageSpec) Data {
	t.Helper()
	d := Data{
		SiteName:          p.SiteName,
		SiteHeadline:      p.SiteHeadline,
		SiteTitle:         p.SiteTitle,
		SiteIcon:          p.SiteIcon,
		OGTitle:           p.OGTitle,
		OGDescription:     p.OGDescription,
		OGImage:           p.OGImage,
		OGURL:             p.OGURL,
		NoIndex:           p.NoIndex,
		GoogleTagID:       p.GoogleTagID,
		ThemeURL:          p.ThemeURL,
		ComputedStyle:     p.ComputedStyle,
		NavTop:            p.NavTop,
		NavSide:           p.NavSide,
		Title:             p.Title,
		Content:           p.Content,
		RevNumber:         p.RevNumber,
		LoginStatusConfig: p.LoginStatusConfig,
		OptionsConfig:     p.OptionsConfig,
	}
	for _, c := range p.Breadcrumbs {
		d.Breadcrumbs = append(d.Breadcrumbs, Breadcrumb{URL: c.URL, Title: c.Title})
	}
	for _, c := range p.TagCategories {
		cat := TagCategory{Name: c.Name}
		for _, tag := range c.Tags {
			cat.Tags = append(cat.Tags, Tag{Name: tag.Name, FullName: tag.FullName})
		}
		d.TagCategories = append(d.TagCategories, cat)
	}
	if p.UpdatedAt != "" {
		at, err := time.Parse(time.RFC3339, p.UpdatedAt)
		if err != nil {
			t.Fatalf("Parse(%q) err = %v, want nil", p.UpdatedAt, err)
		}
		d.UpdatedAt = at
	}
	return d
}

func render(t *testing.T, r *Renderer, c caseSpec) string {
	t.Helper()
	switch c.Kind {
	case "page":
		var b strings.Builder
		if err := r.Page(&b, dataFor(t, c.Page)); err != nil {
			t.Fatalf("Page(%s) err = %v, want nil", c.Name, err)
		}
		return b.String()
	case "not_found":
		html, err := r.NotFound(NotFound{PageID: c.PageID, AllowCreate: c.AllowCreate, Options: c.Options})
		if err != nil {
			t.Fatalf("NotFound(%s) err = %v, want nil", c.Name, err)
		}
		return html
	case "forbidden":
		html, err := r.Forbidden(c.PageID)
		if err != nil {
			t.Fatalf("Forbidden(%s) err = %v, want nil", c.Name, err)
		}
		return html
	}
	t.Fatalf("case %s has kind %q, want one of page, not_found, forbidden", c.Name, c.Kind)
	return ""
}

func TestRenderMatchesGolden(t *testing.T) {
	c := corpus()
	r := testRenderer(t)

	var b strings.Builder
	for _, spec := range c.Cases {
		fmt.Fprintf(&b, "=== %s\n%s\n", spec.Name, render(t, r, spec))
	}
	got := b.String()

	if *update {
		writeCorpus(t, c)
		if err := os.WriteFile(filepath.FromSlash(goldenPath), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(filepath.FromSlash(goldenPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", goldenPath, err)
	}
	if got != string(want) {
		gotAt, wantAt := firstDiff(got, string(want))
		t.Errorf("render = %q, want %q", gotAt, wantAt)
	}
}

func firstDiff(got, want string) (string, string) {
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			return excerpt(got, i), excerpt(want, i)
		}
	}
	return excerpt(got, min(len(got), len(want))), excerpt(want, min(len(got), len(want)))
}

func excerpt(s string, at int) string {
	start := max(0, at-40)
	end := min(len(s), at+40)
	return s[start:end]
}

func writeCorpus(t *testing.T, c corpusFile) {
	t.Helper()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(corpus) err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.FromSlash(corpusPath), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", corpusPath, err)
	}
}

func profileTestRenderer(t *testing.T) *Renderer {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return New(bundle.Localizer(i18n.DefaultLanguage), static.NewAssets(nil), time.UTC)
}

func TestProfileFeedRendersOneRowPerItem(t *testing.T) {
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	got, err := profileTestRenderer(t).Profile(Profile{
		DisplayName: "probe-author",
		JoinedAt:    at,
		Edits: ProfileFeed{Items: []ProfileItem{
			{URL: "/scp-173", Title: "SCP-173", Site: "Test Wiki", At: at,
				Flags: []ProfileFlag{{ID: "S", Desc: "source"}}, Comment: "typo"},
			{URL: "/component:box", Title: "Box", Site: "Test Wiki", At: at},
		}},
	})
	if err != nil {
		t.Fatalf("Profile() err = %v, want nil", err)
	}
	if n := strings.Count(got, `<a class="what"`); n != 2 {
		t.Errorf("count of feed links = %d, want 2", n)
	}
	if !strings.Contains(got, `href="/component:box">Box</a>`) {
		t.Error(`Profile() does not link "Box" to /component:box`)
	}
	if !strings.Contains(got, `>Test Wiki<`) {
		t.Error(`Profile() does not name the site a row came from`)
	}
	if !strings.Contains(got, `<span class="spantip" title="source">S</span>`) {
		t.Error(`Profile() does not flag what an edit changed`)
	}
	if !strings.Contains(got, "typo") {
		t.Error(`Profile() does not show an edit comment`)
	}
}

func TestProfileFeedFallsBackToTheEmptyLine(t *testing.T) {
	got, err := profileTestRenderer(t).Profile(Profile{DisplayName: "probe-author"})
	if err != nil {
		t.Fatalf("Profile() err = %v, want nil", err)
	}
	if n := strings.Count(got, `class="empty"`); n != 2 {
		t.Errorf("count of empty feeds = %d, want 2", n)
	}
	if strings.Contains(got, `<ul class="feed">`) {
		t.Error("Profile() renders a feed list with no items")
	}
}

func TestProfileEditCarriesTheToken(t *testing.T) {
	got, err := profileTestRenderer(t).ProfileEdit(ProfileEdit{
		DisplayName: "probe-author", CSRF: "tok", FullName: "Ada Lovelace",
	})
	if err != nil {
		t.Fatalf("ProfileEdit() err = %v, want nil", err)
	}
	if !strings.Contains(got, `name="csrfmiddlewaretoken" value="tok"`) {
		t.Error("ProfileEdit() does not carry the csrf token")
	}
	if !strings.Contains(got, `enctype="multipart/form-data"`) {
		t.Error("ProfileEdit() posts a form that cannot carry a file")
	}
	if strings.Contains(got, `class="error-inline"`) {
		t.Error("ProfileEdit() shows an error block with no error")
	}
}

func TestProfileEditShowsTheProblem(t *testing.T) {
	got, err := profileTestRenderer(t).ProfileEdit(ProfileEdit{
		DisplayName: "probe-author", Error: "too big",
	})
	if err != nil {
		t.Fatalf("ProfileEdit() err = %v, want nil", err)
	}
	if !strings.Contains(got, "<span>too big</span>") {
		t.Error("ProfileEdit() does not show the error it was given")
	}
}
