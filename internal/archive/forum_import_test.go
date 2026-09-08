package archive

import (
	"testing"
)

func TestFlattenNamesEveryParentAheadOfItsChildren(t *testing.T) {
	tree := []Post{
		{ID: 1, Title: "a", Children: []Post{
			{ID: 2, Title: "b", Children: []Post{{ID: 3, Title: "c"}}},
			{ID: 4, Title: "d"},
		}},
		{ID: 5, Title: "e"},
	}
	im := &importer{}
	posts, parents := im.flatten(tree, -1, nil, nil, map[string]string{})

	wantNames := []string{"a", "b", "c", "d", "e"}
	if len(posts) != len(wantNames) {
		t.Fatalf("len(flatten) = %d, want %d", len(posts), len(wantNames))
	}
	for i, want := range wantNames {
		if posts[i].Name != want {
			t.Errorf("flatten()[%d].Name = %q, want %q", i, posts[i].Name, want)
		}
	}
	wantParents := []int{-1, 0, 1, 0, -1}
	for i, want := range wantParents {
		if parents[i] != want {
			t.Errorf("flatten() parent of %q = %d, want %d", posts[i].Name, parents[i], want)
		}
		if parents[i] >= i {
			t.Errorf("flatten() parent of %q = %d, want less than %d", posts[i].Name, parents[i], i)
		}
	}
}

func TestVersionsReadOldestFirst(t *testing.T) {
	post := Post{
		ID:    7,
		Stamp: 100,
		Revisions: []PostRevision{
			{ID: 30, Stamp: 300},
			{ID: 20, Stamp: 200},
			{ID: 10, Stamp: 100},
		},
	}
	bodies := map[string]string{
		"7/10.html":     "<p>one</p>",
		"7/20.html":     "<p>two</p>",
		"7/30.html":     "<p>three</p>",
		"7/latest.html": "<p>three</p>",
	}
	im := &importer{}
	got := im.versions(post, bodies)

	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("len(versions) = %d, want %d", len(got), len(want))
	}
	for i, text := range want {
		if got[i].Source != text {
			t.Errorf("versions()[%d].Source = %q, want %q", i, got[i].Source, text)
		}
		if got[i].At.Unix() != int64(100*(i+1)) {
			t.Errorf("versions()[%d].At = %d, want %d", i, got[i].At.Unix(), 100*(i+1))
		}
	}
}

func TestVersionsFallsBackToTheLatestBody(t *testing.T) {
	im := &importer{}
	got := im.versions(Post{ID: 7, Stamp: 100}, map[string]string{"7/latest.html": "\n\t<p>only</p>\n"})
	if len(got) != 1 {
		t.Fatalf("len(versions) = %d, want 1", len(got))
	}
	if got[0].Source != "only" {
		t.Errorf("versions()[0].Source = %q, want %q", got[0].Source, "only")
	}
}

func TestStartedFallsBackToTheFirstPoster(t *testing.T) {
	author := int64(11)
	if got := started(Thread{StartedUser: &author}); got != 11 {
		t.Errorf("started(with startedUser) = %d, want 11", got)
	}
	if got := started(Thread{Posts: []Post{{Poster: 22}}}); got != 22 {
		t.Errorf("started(without startedUser) = %d, want 22", got)
	}
	if got := started(Thread{}); got != 0 {
		t.Errorf("started(empty) = %d, want 0", got)
	}
}
