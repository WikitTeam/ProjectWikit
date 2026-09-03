package webapi

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func entry(rev int, kind, meta string) db.LogEntry {
	return db.LogEntry{RevNumber: rev, Type: kind, Meta: []byte(meta)}
}

func TestPlanRevertTakesTheOldestTitleAboveTheTarget(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(3, db.LogTitle, `{"title": "third", "prev_title": "second"}`),
		entry(2, db.LogTitle, `{"title": "second", "prev_title": "first"}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if got.title == nil || *got.title != "first" {
		t.Errorf("planRevert().title = %v, want %q", got.title, "first")
	}
}

func TestPlanRevertStopsAtTheCreation(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(2, db.LogTitle, `{"title": "b", "prev_title": "a"}`),
		entry(1, db.LogNew, `{"title": "a", "version_id": 7}`),
		entry(0, db.LogTitle, `{"title": "a", "prev_title": "never"}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if got.title == nil || *got.title != "a" {
		t.Errorf("planRevert().title = %v, want %q", got.title, "a")
	}
}

func TestPlanRevertSwapsTagsAround(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(1, db.LogTags, `{"added_tags": [{"id": 5, "name": "new"}], "removed_tags": [{"id": 9, "name": "old"}]}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if len(got.removedTags) != 1 || got.removedTags[0] != 5 {
		t.Errorf("planRevert().removedTags = %v, want [5]", got.removedTags)
	}
	if len(got.addedTags) != 1 || got.addedTags[0] != 9 {
		t.Errorf("planRevert().addedTags = %v, want [9]", got.addedTags)
	}
}

func TestPlanRevertLeavesATagOnOneSideOnly(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(2, db.LogTags, `{"added_tags": [], "removed_tags": [{"id": 5, "name": "x"}]}`),
		entry(1, db.LogTags, `{"added_tags": [{"id": 5, "name": "x"}], "removed_tags": []}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if len(got.addedTags) != 0 {
		t.Errorf("planRevert().addedTags = %v, want none", got.addedTags)
	}
	if len(got.removedTags) != 1 || got.removedTags[0] != 5 {
		t.Errorf("planRevert().removedTags = %v, want [5]", got.removedTags)
	}
}

func TestPlanRevertReadsAParentBackToNothing(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(1, db.LogParent, `{"parent": "main", "parent_id": 4, "prev_parent": null, "prev_parent_id": null}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if !got.parentSet {
		t.Fatal("planRevert().parentSet = false, want true")
	}
	if got.parentID != nil {
		t.Errorf("planRevert().parentID = %v, want nil", got.parentID)
	}
}

func TestPlanRevertUnwrapsARevertEntry(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(2, db.LogRevert, `{"rev_number": 0, "subtypes": ["title", "tags"],
			"title": {"title": "b", "prev_title": "a"},
			"tags": {"added": [3], "removed": [4]}}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if got.title == nil || *got.title != "a" {
		t.Errorf("planRevert().title = %v, want %q", got.title, "a")
	}
	if len(got.removedTags) != 1 || got.removedTags[0] != 3 {
		t.Errorf("planRevert().removedTags = %v, want [3]", got.removedTags)
	}
	if len(got.addedTags) != 1 || got.addedTags[0] != 4 {
		t.Errorf("planRevert().addedTags = %v, want [4]", got.addedTags)
	}
}

func TestPlanRevertNamesTheVersionASourceCameFrom(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(2, db.LogSource, `{"version_id": 31}`),
		entry(1, db.LogSource, `{"version_id": 30}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if got.sourceFrom == nil || *got.sourceFrom != 30 {
		t.Errorf("planRevert().sourceFrom = %v, want 30", got.sourceFrom)
	}
}

func TestPlanRevertFlipsAFileAddition(t *testing.T) {
	got, err := planRevert([]db.LogEntry{
		entry(1, db.LogFileAdded, `{"id": 12, "name": "a.png"}`),
	})
	if err != nil {
		t.Fatalf("planRevert() err = %v, want nil", err)
	}
	if !got.filesDeleted[12] {
		t.Error("planRevert().filesDeleted[12] = false, want true")
	}
	if got.filesRestored[12] {
		t.Error("planRevert().filesRestored[12] = true, want false")
	}
}
