package userpage

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	dayParam = "day"
	dayMost  = 300
)

type dayItem struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Site  string `json:"site"`
	At    string `json:"at"`
	Note  string `json:"note,omitempty"`
	at    time.Time
}

func (h *Handler) serveDay(w http.ResponseWriter, r *http.Request, name string) {
	items, err := h.day(r, name)
	if errors.Is(err, db.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		h.log().Error("read profile day", "path", r.URL.Path, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}

func (h *Handler) day(r *http.Request, name string) ([]dayItem, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return nil, errors.New("userpage: the request carries no site")
	}
	viewer := auth.FromContext(ctx)
	loc := h.deps.Bundle.For(ctx)
	profile, err := h.lookup(r, name)
	if err != nil {
		return nil, err
	}
	if !h.showsStats(ctx, profile.ID) {
		return nil, db.ErrNotFound
	}
	zone := site.Zone(ctx)
	start, err := time.ParseInLocation(time.DateOnly, r.URL.Query().Get(dayParam), zone)
	if err != nil {
		return nil, db.ErrNotFound
	}
	end := start.AddDate(0, 0, 1)

	sites, err := h.siteSet(ctx, current)
	if err != nil {
		return nil, err
	}
	span := window{from: &start, before: &end, limit: dayMost}
	var hidden []string
	for i := range sites.all {
		one := &sites.all[i]
		if one.Slug == r.URL.Query().Get(siteParam) {
			span.site = &one.ID
		}
		cats, err := repo.HiddenCategoriesOn(ctx, h.deps.DB, one, viewer)
		if err != nil {
			return nil, err
		}
		for _, c := range cats {
			hidden = append(hidden, strconv.FormatInt(one.ID, 10)+":"+c)
		}
	}

	var out []dayItem
	edits, err := h.edits(r, loc, current, profile, viewer, span)
	if err != nil {
		return nil, err
	}
	for _, e := range edits.Items {
		flags := make([]string, 0, len(e.Flags))
		for _, f := range e.Flags {
			flags = append(flags, f.ID)
		}
		out = append(out, dayItem{Kind: "edit", Title: e.Title, URL: e.URL, Site: e.Site, at: e.At,
			Note: strings.TrimSpace(e.Rev + " " + strings.Join(flags, ""))})
	}
	posts, err := h.posts(r, loc, current, profile, viewer, span, false)
	if err != nil {
		return nil, err
	}
	for _, p := range posts.Items {
		out = append(out, dayItem{Kind: "post", Title: p.Title, URL: p.URL, Site: p.Site, at: p.At})
	}
	votes, err := h.deps.DB.UserVotes(ctx, profile.ID, span.site, start, end, hidden, dayMost)
	if err != nil {
		return nil, err
	}
	for _, v := range votes {
		on := sites.of(v.SiteID)
		article := db.Article{Category: v.Category, Name: v.Name, Title: v.Title}
		out = append(out, dayItem{Kind: "vote", Title: article.DisplayName(), URL: sites.href(on, "/"+article.FullName()),
			Site: on.Title, at: v.At, Note: signed(v.Rate)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at.After(out[j].at) })
	for i := range out {
		out[i].At = out[i].at.In(zone).Format("15:04")
	}
	if out == nil {
		out = []dayItem{}
	}
	return out, nil
}
