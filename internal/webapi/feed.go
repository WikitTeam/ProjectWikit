package webapi

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const FeedPath = listpages.FeedPath

type Feed struct {
	deps Deps
}

var _ http.Handler = (*Feed)(nil)

func NewFeed(d Deps) *Feed { return &Feed{deps: d} }

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// Feed readers ask without signing in, so the list is whatever a visitor who
// is not signed in may see.
func (h *Feed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	loc := h.deps.Bundle.For(ctx)
	var user *db.User

	query := r.URL.Query()
	var article *db.Article
	if name := query.Get("page"); name != "" {
		found, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), name)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			h.deps.log().Error("feed", "page", name, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		article = found
	}
	params := map[string]string{}
	for key, values := range query {
		if key != "page" && len(values) > 0 {
			params[strings.ToLower(key)] = values[0]
		}
	}

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons, Trust: h.deps.Trust}.
		Env(ctx, loc, current, user)
	pc := page.NewContext(article, article, nil, user)
	out, err := env.ModuleAPI(pc, "listpages", "feed", params)
	if err != nil {
		h.deps.log().Error("feed", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	origin := h.deps.Trust.Scheme(r) + "://" + current.Domain
	channel := rssChannel{
		Title:       text(out, "title"),
		Description: text(out, "description"),
		Link:        origin + "/",
	}
	if home := strings.Trim(text(out, "home"), "/ "); home != "" {
		channel.Link = origin + "/" + home
	} else if article != nil {
		channel.Link = origin + "/" + article.FullName()
	}
	if channel.Title == "" {
		channel.Title = current.Title
	}
	if items, ok := lookup(out, "items").(wikijson.Array); ok {
		for _, raw := range items {
			item, ok := raw.(wikijson.Object)
			if !ok {
				continue
			}
			link := origin + "/" + text(item, "name")
			at, _ := lookup(item, "at").(time.Time)
			channel.Items = append(channel.Items, rssItem{
				Title: text(item, "title"), Link: link, GUID: link,
				PubDate: at.UTC().Format(time.RFC1123Z), Description: text(item, "summary"),
			})
		}
	}

	body, err := xml.MarshalIndent(rss{Version: "2.0", Channel: channel}, "", "  ")
	if err != nil {
		h.deps.log().Error("feed", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(xml.Header))
		_, _ = w.Write(body)
	}
}

func lookup(o wikijson.Object, key string) any {
	for _, f := range o {
		if f.Key == key {
			return f.Value
		}
	}
	return nil
}

func text(o wikijson.Object, key string) string {
	s, _ := lookup(o, key).(string)
	return s
}
