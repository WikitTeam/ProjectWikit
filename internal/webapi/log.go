package webapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/changelog"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const defaultLogPage = 25

func (h *Articles) log(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}

	from, to, all, ok := logRange(r)
	if !ok {
		return field("error", loc.T("api-bad-range")), http.StatusBadRequest, nil
	}
	total, err := h.deps.DB.ArticleLogCount(ctx, article.ID)
	if err != nil {
		return "", 0, err
	}
	offset, limit := 0, (*int)(nil)
	if !all {
		offset = from
		span := to - from
		if span < 0 {
			span = 0
		}
		limit = &span
	}
	entries, err := h.deps.DB.ArticleLog(ctx, article.ID, offset, limit)
	if err != nil {
		return "", 0, err
	}

	users := func(ids []int64) ([]db.User, error) { return h.deps.DB.UsersByIDs(ctx, ids) }
	rendered := make(wikijson.Array, 0, len(entries))
	for _, entry := range entries {
		one, err := h.logEntryJSON(r, loc, users, entry)
		if err != nil {
			return "", 0, err
		}
		rendered = append(rendered, one)
	}

	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "count", Value: total},
		{Key: "entries", Value: rendered},
	})
	return body, http.StatusOK, err
}

// A range that is not a pair of numbers is refused rather than quietly replaced
// with the default.
func logRange(r *http.Request) (from, to int, all, ok bool) {
	query := r.URL.Query()
	from, to = 0, defaultLogPage
	if raw := query.Get("from"); raw != "" {
		if from, ok = number(raw); !ok {
			return 0, 0, false, false
		}
	}
	if raw := query.Get("to"); raw != "" {
		if to, ok = number(raw); !ok {
			return 0, 0, false, false
		}
	}
	return from, to, query.Get("all") != "", true
}

func number(raw string) (int, bool) {
	n, err := strconv.Atoi(raw)
	return n, err == nil
}

func (h *Articles) logEntryJSON(r *http.Request, loc *i18n.Localizer, users changelog.Users,
	entry db.LogEntry) (wikijson.Object, error) {

	author, err := h.logEntryUser(r, entry.UserID)
	if err != nil {
		return nil, err
	}
	told, err := changelog.Of(loc, users, db.SiteChange{
		Type: entry.Type, Meta: entry.Meta, Comment: entry.Comment,
	})
	if err != nil {
		return nil, err
	}

	meta, err := decodeMeta(entry.Meta)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "revNumber", Value: entry.RevNumber},
		{Key: "user", Value: author},
		{Key: "comment", Value: entry.Comment},
		{Key: "defaultComment", Value: told.Comment},
		{Key: "createdAt", Value: isoTime(entry.CreatedAt)},
		{Key: "type", Value: entry.Type},
		{Key: "meta", Value: fromJSON(meta)},
	}, nil
}

func (h *Articles) logEntryUser(r *http.Request, id *int64) (wikijson.Object, error) {
	if id == nil {
		return repo.UserJSON(r.Context(), h.deps.DB, nil)
	}
	found, err := h.deps.DB.UserByID(r.Context(), *id)
	if err != nil {
		return nil, err
	}
	return repo.UserJSON(r.Context(), h.deps.DB, found)
}

// Numbers are kept as written rather than turned into floats, so a revision id
// goes back out as the whole number it was stored as.
func decodeMeta(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// Whatever the revision stored goes back out unchanged, so a key this version
// never learned to write still reaches the reader.
func fromJSON(value any) any {
	switch v := value.(type) {
	case json.Number:
		if whole, err := v.Int64(); err == nil {
			return whole
		}
		fraction, err := v.Float64()
		if err != nil {
			return v.String()
		}
		return fraction
	case map[string]any:
		out := make(wikijson.Object, 0, len(v))
		for _, key := range sortedKeys(v) {
			out = append(out, wikijson.Field{Key: key, Value: fromJSON(v[key])})
		}
		return out
	case []any:
		out := make(wikijson.Array, 0, len(v))
		for _, one := range v {
			out = append(out, fromJSON(one))
		}
		return out
	}
	return value
}

// jsonb keeps its own key order, shorter first and then bytewise, and that is
// the order the revision was read out in.
func sortedKeys(fields map[string]any) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})
	return keys
}

func isoTime(at time.Time) string {
	return at.UTC().Format("2006-01-02T15:04:05.999999-07:00")
}
