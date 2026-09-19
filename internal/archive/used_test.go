package archive

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func usedFixture(t *testing.T) *Archive {
	t.Helper()
	dir := site(t, t.TempDir(), "my-wiki")
	writeFile(t, filepath.Join(dir, metaDir, pagesDir, "start.json"), `{
		"name": "start",
		"revisions": [{"revision": 1, "author": 11, "stamp": 1, "flags": "S"}, {"revision": 0, "author": 10, "stamp": 0, "flags": "N"}],
		"votings": [[12, true]],
		"files": [{"file_id": 1, "name": "a.png", "author": 13, "stamp": 0}]
	}`)
	writeFile(t, filepath.Join(dir, metaDir, forumDir, categoryDir, "5.json"), `{"id": 5, "title": "Chat"}`)
	writeFile(t, filepath.Join(dir, metaDir, forumDir, "5", "7.json"), `{
		"id": 7, "startedUser": 14,
		"posts": [{"id": 1, "poster": 15, "revisions": [{"id": 1, "author": 16}], "children": [{"id": 2, "poster": 17}]}]
	}`)
	found, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	return found
}

func usedIDs(t *testing.T, a *Archive, opts Options) []int64 {
	t.Helper()
	pages, err := a.Pages("my-wiki")
	if err != nil {
		t.Fatalf("Pages() err = %v, want nil", err)
	}
	used, err := a.usedUsers("my-wiki", pages, opts)
	if err != nil {
		t.Fatalf("usedUsers() err = %v, want nil", err)
	}
	out := make([]int64, 0, len(used))
	for id := range used {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func TestUsedUsersCountsEveryReference(t *testing.T) {
	got := usedIDs(t, usedFixture(t), Options{Votes: true, Files: "files"})
	want := []int64{10, 11, 12, 13, 14, 15, 16, 17}
	if !slices.Equal(got, want) {
		t.Errorf("usedUsers() = %v, want %v", got, want)
	}
}

func TestUsedUsersSkipsWhatIsLeftBehind(t *testing.T) {
	got := usedIDs(t, usedFixture(t), Options{})
	want := []int64{10, 11, 14, 15, 16, 17}
	if !slices.Equal(got, want) {
		t.Errorf("usedUsers() = %v, want %v", got, want)
	}
}
