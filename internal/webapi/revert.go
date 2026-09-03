package webapi

import (
	"encoding/json"
	"slices"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

// The walk that fills this runs newest first, so the oldest entry above the
// target is the one whose value survives.
type plan struct {
	sourceFrom *int64
	title      *string
	name       *string

	parentID  *int64
	parentSet bool

	addedTags   []int64
	removedTags []int64

	addedAuthors   []int64
	removedAuthors []int64

	filesDeleted  map[int64]bool
	filesRestored map[int64]bool
	filesRenamed  map[int64]string

	votes json.RawMessage
}

func newPlan() *plan {
	return &plan{
		filesDeleted:  map[int64]bool{},
		filesRestored: map[int64]bool{},
		filesRenamed:  map[int64]string{},
	}
}

// A page's creation cannot be undone, so the walk stops rather than trying.
func planRevert(entries []db.LogEntry) (*plan, error) {
	p := newPlan()
	for _, entry := range entries {
		meta, err := decodeMeta(entry.Meta)
		if err != nil {
			return nil, err
		}
		fields, _ := meta.(map[string]any)
		if entry.Type == db.LogNew {
			break
		}
		p.undo(entry.Type, fields)
	}
	return p, nil
}

func (p *plan) undo(kind string, meta map[string]any) {
	switch kind {
	case db.LogSource:
		p.sourceFrom = wholeAt(meta, "version_id")
	case db.LogTitle:
		p.title = textAt(meta, "prev_title")
	case db.LogName:
		p.name = textAt(meta, "prev_name")
	case db.LogParent:
		p.parentID, p.parentSet = wholeAt(meta, "prev_parent_id"), true
	case db.LogTags:
		p.swapTags(namedIDs(meta, "added_tags"), namedIDs(meta, "removed_tags"))
	case db.LogAuthorship:
		p.swapAuthors(wholeList(meta, "added_authors"), wholeList(meta, "removed_authors"))
	case db.LogVotesDeleted:
		p.votes, _ = json.Marshal(meta)
	case db.LogFileAdded:
		p.markFile(meta, true, false)
	case db.LogFileDeleted:
		p.markFile(meta, false, true)
	case db.LogFileRenamed:
		if id := wholeAt(meta, "id"); id != nil {
			if previous := textAt(meta, "prev_name"); previous != nil {
				p.filesRenamed[*id] = *previous
			}
		}
	case db.LogRevert:
		p.undoRevert(meta)
	}
}

// A revert entry keeps what it undid under a key per kind, so undoing one is
// the same work read out of a different shape.
func (p *plan) undoRevert(meta map[string]any) {
	if inner := objectAt(meta, "source"); inner != nil {
		p.sourceFrom = wholeAt(inner, "version_id")
	}
	if inner := objectAt(meta, "title"); inner != nil {
		p.title = textAt(inner, "prev_title")
	}
	if inner := objectAt(meta, "name"); inner != nil {
		p.name = textAt(inner, "prev_name")
	}
	if inner := objectAt(meta, "parent"); inner != nil {
		p.parentID, p.parentSet = wholeAt(inner, "prev_parent_id"), true
	}
	if inner := objectAt(meta, "tags"); inner != nil {
		p.swapTags(wholeList(inner, "added"), wholeList(inner, "removed"))
	}
	if inner := objectAt(meta, "authorship"); inner != nil {
		p.swapAuthors(wholeList(inner, "added"), wholeList(inner, "removed"))
	}
	if inner := objectAt(meta, "files"); inner != nil {
		for _, one := range objectList(inner, "added") {
			p.markFile(one, true, false)
		}
		for _, one := range objectList(inner, "deleted") {
			p.markFile(one, false, true)
		}
		for _, one := range objectList(inner, "renamed") {
			if id := wholeAt(one, "id"); id != nil {
				if previous := textAt(one, "prev_name"); previous != nil {
					p.filesRenamed[*id] = *previous
				}
			}
		}
	}
	if inner := objectAt(meta, "votes"); inner != nil {
		p.votes, _ = json.Marshal(inner)
	}
}

func (p *plan) markFile(meta map[string]any, deleted, restored bool) {
	id := wholeAt(meta, "id")
	if id == nil {
		return
	}
	p.filesDeleted[*id] = deleted
	p.filesRestored[*id] = restored
}

// What an entry added is now removed and what it removed is now added, so each
// id moves to the other list rather than joining both.
func (p *plan) swapTags(added, removed []int64) {
	p.addedTags, p.removedTags = swapped(p.addedTags, p.removedTags, added, removed)
}

func (p *plan) swapAuthors(added, removed []int64) {
	p.addedAuthors, p.removedAuthors = swapped(p.addedAuthors, p.removedAuthors, added, removed)
}

func swapped(added, removed, entryAdded, entryRemoved []int64) ([]int64, []int64) {
	for _, id := range entryAdded {
		added = dropFirst(added, id)
		removed = append(removed, id)
	}
	for _, id := range entryRemoved {
		removed = dropFirst(removed, id)
		added = append(added, id)
	}
	return added, removed
}

func dropFirst(list []int64, id int64) []int64 {
	if at := slices.Index(list, id); at >= 0 {
		return slices.Delete(list, at, at+1)
	}
	return list
}

func textAt(meta map[string]any, key string) *string {
	if value, ok := meta[key].(string); ok {
		return &value
	}
	return nil
}

func wholeAt(meta map[string]any, key string) *int64 {
	number, ok := meta[key].(json.Number)
	if !ok {
		return nil
	}
	value, err := number.Int64()
	if err != nil {
		return nil
	}
	return &value
}

func objectAt(meta map[string]any, key string) map[string]any {
	inner, _ := meta[key].(map[string]any)
	return inner
}

func wholeList(meta map[string]any, key string) []int64 {
	raw, _ := meta[key].([]any)
	out := make([]int64, 0, len(raw))
	for _, one := range raw {
		if number, ok := one.(json.Number); ok {
			if value, err := number.Int64(); err == nil {
				out = append(out, value)
			}
		}
	}
	return out
}

func objectList(meta map[string]any, key string) []map[string]any {
	raw, _ := meta[key].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, one := range raw {
		if inner, ok := one.(map[string]any); ok {
			out = append(out, inner)
		}
	}
	return out
}

func namedIDs(meta map[string]any, key string) []int64 {
	out := make([]int64, 0)
	for _, one := range objectList(meta, key) {
		if id := wholeAt(one, "id"); id != nil {
			out = append(out, *id)
		}
	}
	return out
}
