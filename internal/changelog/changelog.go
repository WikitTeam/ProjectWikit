// Package changelog turns one article log entry into the flags and comment a
// reader sees beside it.
package changelog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

var ErrUnreadable = errors.New("changelog: the log entry cannot be read")

type Users func(ids []int64) ([]db.User, error)

type Flag struct {
	ID   string
	Desc string
}

type Entry struct {
	Flags   []Flag
	Comment string
}

var flags = map[string]string{
	"source": "S", "title": "T", "name": "R", "tags": "A", "new": "N",
	"parent": "M", "file_added": "F", "file_deleted": "F", "file_renamed": "F",
	"votes_deleted": "V", "authorship": "C", "wikidot": "W",
}

func TypeName(loc *i18n.Localizer, t string) (flag, desc string) {
	f, ok := flags[t]
	if !ok {
		return "?", "?"
	}
	return f, text(loc, "module-sitechanges-type-"+strings.ReplaceAll(t, "_", "-"))
}

// A revert's flags are the types it undid, which it keeps in meta rather than
// in the type column.
func Of(loc *i18n.Localizer, users Users, change db.SiteChange) (Entry, error) {
	m, err := metaOf(change.Meta)
	if err != nil {
		return Entry{}, err
	}

	var entry Entry
	if raw, ok := m["subtypes"]; ok {
		var subtypes []string
		if err := json.Unmarshal(raw, &subtypes); err != nil {
			return Entry{}, ErrUnreadable
		}
		for _, subtype := range subtypes {
			id, desc := TypeName(loc, subtype)
			entry.Flags = append(entry.Flags, Flag{ID: id, Desc: desc})
		}
	} else {
		id, desc := TypeName(loc, change.Type)
		entry.Flags = append(entry.Flags, Flag{ID: id, Desc: desc})
	}

	if entry.Comment, err = comment(loc, users, change, m); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func comment(loc *i18n.Localizer, users Users, change db.SiteChange, m metaMap) (string, error) {
	if strings.TrimSpace(change.Comment) != "" {
		return change.Comment, nil
	}

	switch change.Type {
	case "new":
		return text(loc, "module-sitechanges-comment-new"), nil
	case "title":
		prev, title, err := m.two("prev_title", "title")
		if err != nil {
			return "", err
		}
		return text(loc, "module-sitechanges-comment-title", "prev", prev, "title", title), nil
	case "name":
		prev, name, err := m.two("prev_name", "name")
		if err != nil {
			return "", err
		}
		return text(loc, "module-sitechanges-comment-name", "prev", prev, "name", name), nil
	case "tags":
		return tagComment(loc, m)
	case "parent":
		return parentComment(loc, m)
	case "file_added":
		name, err := m.str("name")
		if err != nil {
			return "", err
		}
		return text(loc, "module-sitechanges-comment-file-added", "name", name), nil
	case "file_deleted":
		name, err := m.str("name")
		if err != nil {
			return "", err
		}
		return text(loc, "module-sitechanges-comment-file-deleted", "name", name), nil
	case "file_renamed":
		prev, name, err := m.two("prev_name", "name")
		if err != nil {
			return "", err
		}
		return text(loc, "module-sitechanges-comment-file-renamed", "prev", prev, "name", name), nil
	case "votes_deleted":
		return votesComment(loc, m)
	case "authorship":
		return authorComment(loc, users, m)
	case "revert":
		rev, err := m.str("rev_number")
		if err != nil {
			return "", err
		}
		return text(loc, "module-sitechanges-comment-revert", "rev", rev), nil
	}
	return "", nil
}

func tagComment(loc *i18n.Localizer, m metaMap) (string, error) {
	added, err := m.names("added_tags")
	if err != nil {
		return "", err
	}
	removed, err := m.names("removed_tags")
	if err != nil {
		return "", err
	}

	var parts []string
	if len(added) > 0 {
		parts = append(parts, text(loc, "module-sitechanges-comment-tags-added",
			"tags", strings.Join(added, ", ")))
	}
	if len(removed) > 0 {
		parts = append(parts, text(loc, "module-sitechanges-comment-tags-removed",
			"tags", strings.Join(removed, ", ")))
	}
	return strings.Join(parts, " "), nil
}

func parentComment(loc *i18n.Localizer, m metaMap) (string, error) {
	prev, parent, err := m.two("prev_parent", "parent")
	if err != nil {
		return "", err
	}
	switch {
	case m.truthy("prev_parent") && m.truthy("parent"):
		return text(loc, "module-sitechanges-comment-parent-changed", "prev", prev, "parent", parent), nil
	case m.truthy("prev_parent"):
		return text(loc, "module-sitechanges-comment-parent-removed", "prev", prev), nil
	case m.truthy("parent"):
		return text(loc, "module-sitechanges-comment-parent-set", "parent", parent), nil
	}
	return "", nil
}

func votesComment(loc *i18n.Localizer, m metaMap) (string, error) {
	mode, err := m.str("rating_mode")
	if err != nil {
		return "", err
	}
	votes, err := m.str("votes_count")
	if err != nil {
		return "", err
	}
	popularity, err := m.str("popularity")
	if err != nil {
		return "", err
	}

	rating := text(loc, "module-sitechanges-comment-votes-none")
	switch mode {
	case "updown":
		n, err := m.integer("rating")
		if err != nil {
			return "", err
		}
		rating = fmt.Sprintf("%+d", n)
	case "stars":
		f, err := m.float("rating")
		if err != nil {
			return "", err
		}
		rating = strconv.FormatFloat(f, 'f', 1, 64)
	}
	return text(loc, "module-sitechanges-comment-votes",
		"rating", rating, "votes", votes, "popularity", popularity), nil
}

func authorComment(loc *i18n.Localizer, users Users, m metaMap) (string, error) {
	label := func(key string) (string, error) {
		ids, err := m.ids(key)
		if err != nil {
			return "", err
		}
		found, err := users(ids)
		if err != nil {
			return "", err
		}
		names := make([]string, 0, len(found))
		for i := range found {
			names = append(names, found[i].DisplayLabel())
		}
		return strings.Join(names, ", "), nil
	}

	added, err := label("added_authors")
	if err != nil {
		return "", err
	}
	removed, err := label("removed_authors")
	if err != nil {
		return "", err
	}

	var parts []string
	if added != "" {
		parts = append(parts, text(loc, "module-sitechanges-comment-authors-added", "names", added))
	}
	if removed != "" {
		parts = append(parts, text(loc, "module-sitechanges-comment-authors-removed", "names", removed))
	}
	return strings.Join(parts, " "), nil
}

func text(loc *i18n.Localizer, id string, args ...any) string {
	if loc == nil {
		return id
	}
	return loc.T(id, args...)
}

type metaMap map[string]json.RawMessage

func metaOf(raw []byte) (metaMap, error) {
	if len(raw) == 0 {
		return metaMap{}, nil
	}
	var m metaMap
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ErrUnreadable
	}
	return m, nil
}

func (m metaMap) str(key string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", ErrUnreadable
	}
	return metaText(raw), nil
}

func (m metaMap) two(first, second string) (string, string, error) {
	a, err := m.str(first)
	if err != nil {
		return "", "", err
	}
	b, err := m.str(second)
	if err != nil {
		return "", "", err
	}
	return a, b, nil
}

func (m metaMap) truthy(key string) bool {
	switch t := strings.TrimSpace(string(m[key])); t {
	case "", "null", "false", "0", "0.0", `""`, "[]", "{}":
		return false
	}
	return true
}

func (m metaMap) names(key string) ([]string, error) {
	raw, ok := m[key]
	if !ok {
		return nil, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, ErrUnreadable
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		name, ok := item["name"]
		if !ok {
			return nil, ErrUnreadable
		}
		out = append(out, metaText(name))
	}
	return out, nil
}

func (m metaMap) ids(key string) ([]int64, error) {
	raw, ok := m[key]
	if !ok {
		return nil, nil
	}
	var ids []int64
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, ErrUnreadable
	}
	return ids, nil
}

func (m metaMap) integer(key string) (int, error) {
	raw, ok := m[key]
	if !ok {
		return 0, ErrUnreadable
	}
	if quoted, ok := unquoteJSON(raw); ok {
		n, err := wikinum.Int(quoted)
		if err != nil {
			return 0, ErrUnreadable
		}
		return n, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return 0, ErrUnreadable
	}
	return int(f), nil
}

func (m metaMap) float(key string) (float64, error) {
	raw, ok := m[key]
	if !ok {
		return 0, ErrUnreadable
	}
	if quoted, ok := unquoteJSON(raw); ok {
		f, err := wikinum.Float(quoted)
		if err != nil {
			return 0, ErrUnreadable
		}
		return f, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return 0, ErrUnreadable
	}
	return f, nil
}

func unquoteJSON(raw json.RawMessage) (string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(string(raw)), `"`) {
		return "", false
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false
	}
	return out, true
}

// The comment lines interpolate whatever the log entry stored, so a value that
// is not a string still has to come out as text.
func metaText(raw json.RawMessage) string {
	t := strings.TrimSpace(string(raw))
	switch t {
	case "null":
		return "None"
	case "true":
		return "True"
	case "false":
		return "False"
	}
	if unquoted, ok := unquoteJSON(raw); ok {
		return unquoted
	}
	return t
}
