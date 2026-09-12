package archive

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func site(t *testing.T, root, slug string) string {
	t.Helper()
	dir := filepath.Join(root, slug)
	if err := os.MkdirAll(filepath.Join(dir, metaDir, pagesDir), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"slug":"` + slug + `","domain":"` + slug + `.wikidot.com","home_page":"start","site_id":7}`
	if err := os.WriteFile(filepath.Join(dir, metaDir, siteFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestOpenReadsOneSiteDirectory(t *testing.T) {
	root := t.TempDir()
	dir := site(t, root, "my-wiki")

	found, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(a site directory) err = %v, want nil", err)
	}
	if got := found.Sites(); !slices.Equal(got, []string{"my-wiki"}) {
		t.Errorf("Sites() = %v, want [my-wiki]", got)
	}
	meta, err := found.Site("my-wiki")
	if err != nil {
		t.Fatalf("Site() err = %v, want nil", err)
	}
	if meta.Domain != "my-wiki.wikidot.com" {
		t.Errorf("Site().Domain = %q, want %q", meta.Domain, "my-wiki.wikidot.com")
	}
}

func TestOpenReadsTheDirectoryAboveSeveralSites(t *testing.T) {
	root := t.TempDir()
	site(t, root, "second")
	site(t, root, "first")
	if err := os.MkdirAll(filepath.Join(root, usersDir), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"1":{"user_id":1,"username":"someone","full_name":"Some One","fetched_at":10}}`
	if err := os.WriteFile(filepath.Join(root, usersDir, "1.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := Open(root)
	if err != nil {
		t.Fatalf("Open(a directory of sites) err = %v, want nil", err)
	}
	if got := found.Sites(); !slices.Equal(got, []string{"first", "second"}) {
		t.Errorf("Sites() = %v, want [first second]", got)
	}
	users, err := found.Users()
	if err != nil {
		t.Fatalf("Users() err = %v, want nil", err)
	}
	if users[1].Username != "someone" {
		t.Errorf("Users()[1].Username = %q, want %q", users[1].Username, "someone")
	}
}

func TestOpenRefusesAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "my-wiki.tar.gz")
	if err := os.WriteFile(path, []byte("not unpacked"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Open(path)
	if err == nil {
		t.Fatal("Open(a file) err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unpack") {
		t.Errorf("Open(a file) err = %v, want it to say to unpack the backup", err)
	}
}

func TestOpenRefusesADirectoryWithNoSite(t *testing.T) {
	_, err := Open(t.TempDir())
	if err == nil {
		t.Fatal("Open(an empty directory) err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), filepath.Join(metaDir, siteFile)) {
		t.Errorf("Open(an empty directory) err = %v, want it to name %q", err, filepath.Join(metaDir, siteFile))
	}
}
