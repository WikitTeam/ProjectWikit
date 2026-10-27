// Package sitemap answers /sitemap.xml and /robots.txt.
package sitemap

import (
	"context"
	"encoding/xml"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	Path       = "/sitemap.xml"
	RobotsPath = "/robots.txt"
)

const maxPerFile = 50000

var structural = []string{"nav", "admin"}

type Handler struct {
	db  *db.DB
	log *slog.Logger
}

func New(d *db.DB, log *slog.Logger) *Handler { return &Handler{db: d, log: log} }

type entry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name `xml:"urlset"`
	NS      string   `xml:"xmlns,attr"`
	URLs    []entry  `xml:"url"`
}

type index struct {
	XMLName  xml.Name `xml:"sitemapindex"`
	NS       string   `xml:"xmlns,attr"`
	Sitemaps []entry  `xml:"sitemap"`
}

const namespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	current := site.FromContext(r.Context())
	if current == nil {
		http.NotFound(w, r)
		return
	}
	base := "https://" + current.Domain
	if r.URL.Path == RobotsPath {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write([]byte("User-agent: *\nDisallow:\n\nSitemap: " + base + Path + "\n"))
		return
	}

	entries, err := h.entries(r.Context(), current, base)
	if err != nil {
		h.log.Error("pwikit could not build the sitemap", "site", current.Slug, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	doc, ok := document(entries, base, r.URL.Query().Get("part"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	out, err := xml.Marshal(doc)
	if err != nil {
		h.log.Error("pwikit could not write the sitemap", "site", current.Slug, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
}

func document(entries []entry, base, part string) (any, bool) {
	if part != "" {
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || (n-1)*maxPerFile >= len(entries) {
			return nil, false
		}
		return urlSet{NS: namespace, URLs: entries[(n-1)*maxPerFile : min(n*maxPerFile, len(entries))]}, true
	}
	if len(entries) <= maxPerFile {
		return urlSet{NS: namespace, URLs: entries}, true
	}
	idx := index{NS: namespace}
	for n := 1; (n-1)*maxPerFile < len(entries); n++ {
		idx.Sitemaps = append(idx.Sitemaps, entry{Loc: base + Path + "?part=" + strconv.Itoa(n)})
	}
	return idx, true
}

func (h *Handler) entries(ctx context.Context, current *db.Site, base string) ([]entry, error) {
	hidden, err := repo.HiddenCategoriesOn(ctx, h.db, current, nil)
	if err != nil {
		return nil, err
	}
	articles, err := h.db.SitemapArticles(ctx, current.ID, append(hidden, structural...))
	if err != nil {
		return nil, err
	}
	sections, err := h.readableSections(ctx, current)
	if err != nil {
		return nil, err
	}
	threads, err := h.db.SitemapThreads(ctx, current.ID, sections)
	if err != nil {
		return nil, err
	}

	out := make([]entry, 0, len(articles)+len(threads))
	for i := range articles {
		out = append(out, entry{Loc: base + "/" + articles[i].FullName(), LastMod: lastMod(articles[i].UpdatedAt)})
	}
	for _, t := range threads {
		loc := base + "/forum/t-" + strconv.FormatInt(t.ID, 10) + "/" + wikidot.Normalize(t.Name)
		out = append(out, entry{Loc: loc, LastMod: lastMod(t.UpdatedAt)})
	}
	return out, nil
}

// A hidden section is still readable, but listing it would undo the hiding.
func (h *Handler) readableSections(ctx context.Context, current *db.Site) ([]int64, error) {
	resolver := repo.NewPermsOn(ctx, h.db, current)
	subject, err := resolver.Subject(nil, time.Now())
	if err != nil {
		return nil, err
	}
	sitewide := perms.Resolve(subject, nil)
	if !sitewide.Has(perms.ViewForumCategories) || !sitewide.Has(perms.ViewForumThreads) {
		return nil, nil
	}
	sections, err := h.db.ForumSections(ctx, current.ID)
	if err != nil {
		return nil, err
	}
	var out []int64
	for i := range sections {
		s := &sections[i]
		if s.IsHidden || !perms.Resolve(subject, resolver.ForumSection(s)).Has(perms.ViewForumSections) {
			continue
		}
		out = append(out, s.ID)
	}
	return out, nil
}

func lastMod(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
