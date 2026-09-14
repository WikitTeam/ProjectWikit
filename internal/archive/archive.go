// Package archive reads the backup a wikitCLI run leaves behind.
package archive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	metaDir   = "meta"
	pagesDir  = "pages"
	filesDir  = "files"
	usersDir  = "_users"
	siteFile  = "site.json"
	pageSufix = ".json"
	sourceExt = ".7z"
)

type Archive struct {
	// sites maps each slug to the directory holding that site's meta, pages,
	// files and forum.
	sites map[string]string
	// users are the _users directories that belong to no single site.
	users []string
}

type SiteMeta struct {
	Slug     string `json:"slug"`
	Domain   string `json:"domain"`
	HomePage string `json:"home_page"`
	Language string `json:"language"`
	SiteID   int64  `json:"site_id"`
}

type User struct {
	ID        int64  `json:"user_id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	FetchedAt int64  `json:"fetched_at"`
}

type Revision struct {
	Number  int    `json:"revision"`
	Author  int64  `json:"author"`
	Stamp   int64  `json:"stamp"`
	Flags   string `json:"flags"`
	Comment string `json:"commentary"`
}

// HasSource reports whether the 7z holds the wikitext of this revision. Wikidot
// records a revision for a tag or a rename too, and those carry no text.
func (r Revision) HasSource() bool {
	return strings.ContainsAny(r.Flags, "SN")
}

func (r Revision) IsNew() bool { return strings.Contains(r.Flags, "N") }

type File struct {
	ID     int64  `json:"file_id"`
	Name   string `json:"name"`
	Mime   string `json:"mime"`
	Size   int64  `json:"size_bytes"`
	Author int64  `json:"author"`
	Stamp  int64  `json:"stamp"`
}

type Vote struct {
	UserID int64
	Value  float64
}

// UnmarshalJSON reads the [user, value] pair the backup writes. A boolean is the
// up and down scale, a number is the star one.
func (v *Vote) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) < 2 {
		return fmt.Errorf("vote %s has %d fields, want 2", data, len(raw))
	}
	if err := json.Unmarshal(raw[0], &v.UserID); err != nil {
		return err
	}
	var flag bool
	if err := json.Unmarshal(raw[1], &flag); err == nil {
		v.Value = 1
		if !flag {
			v.Value = -1
		}
		return nil
	}
	return json.Unmarshal(raw[1], &v.Value)
}

type Page struct {
	// Stem is the name the meta and the source archive share, which is not the
	// page name and is the only way back to the 7z.
	Stem string `json:"-"`

	Name      string     `json:"name"`
	Title     string     `json:"title"`
	Parent    string     `json:"parent"`
	Tags      []string   `json:"tags"`
	Rating    float64    `json:"rating"`
	Locked    bool       `json:"is_locked"`
	PageID    int64      `json:"page_id"`
	ThreadID  int64      `json:"forum_thread"`
	Revisions []Revision `json:"revisions"`
	Votings   []Vote     `json:"votings"`
	Files     []File     `json:"files"`
}

// The path is either a site directory, recognised by the site meta inside it,
// or a directory holding several of them.
func Open(path string) (*Archive, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is a file, and pwikit reads the unpacked backup; unpack it and pass the directory", path)
	}
	a := &Archive{sites: map[string]string{}}
	if isSite(path) {
		a.sites[filepath.Base(path)] = path
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || e.Name() == usersDir {
				continue
			}
			if dir := filepath.Join(path, e.Name()); isSite(dir) {
				a.sites[e.Name()] = dir
			}
		}
	}
	if len(a.sites) == 0 {
		return nil, fmt.Errorf("no site under %q; a site directory holds %s, and the directory above several of them works too",
			path, filepath.Join(metaDir, siteFile))
	}
	if _, err := os.Stat(filepath.Join(path, usersDir)); err == nil {
		a.users = append(a.users, filepath.Join(path, usersDir))
	}
	return a, nil
}

func isSite(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, metaDir, siteFile))
	return err == nil
}

func (a *Archive) find(slug string, parts ...string) (string, bool) {
	base, ok := a.sites[slug]
	if !ok {
		return "", false
	}
	full := filepath.Join(append([]string{base}, parts...)...)
	if _, err := os.Stat(full); err != nil {
		return "", false
	}
	return full, true
}

// Sites lists the slugs the backup carries. A backup of one site holds one.
func (a *Archive) Sites() []string {
	out := make([]string, 0, len(a.sites))
	for slug := range a.sites {
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

func (a *Archive) Site(slug string) (SiteMeta, error) {
	var s SiteMeta
	path, ok := a.find(slug, metaDir, siteFile)
	if !ok {
		return s, fmt.Errorf("no site %q in the archive", slug)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(body, &s); err != nil {
		return s, fmt.Errorf("read the site meta of %q: %w", slug, err)
	}
	return s, nil
}

// Users merges every _users directory the backup carries, the shared one and
// the one a site brought along. Where two disagree the newer fetch wins.
func (a *Archive) Users() (map[int64]User, error) {
	dirs := append([]string{}, a.users...)
	for _, slug := range a.Sites() {
		dirs = append(dirs, filepath.Join(a.sites[slug], usersDir))
	}

	out := map[int64]User{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), pageSufix) || e.Name() == "pending.json" {
				continue
			}
			body, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				return nil, err
			}
			var bucket map[string]User
			if err := json.Unmarshal(body, &bucket); err != nil {
				return nil, fmt.Errorf("read users from %q: %w", e.Name(), err)
			}
			for _, u := range bucket {
				if u.ID == 0 {
					continue
				}
				if held, ok := out[u.ID]; ok && held.FetchedAt >= u.FetchedAt {
					continue
				}
				out[u.ID] = u
			}
		}
	}
	return out, nil
}

func (a *Archive) Pages(slug string) ([]Page, error) {
	dir, ok := a.find(slug, metaDir, pagesDir)
	if !ok {
		return nil, fmt.Errorf("no pages for %q in the archive", slug)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Page, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), pageSufix) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var p Page
		if err := json.Unmarshal(body, &p); err != nil {
			return nil, fmt.Errorf("read the page meta %q: %w", e.Name(), err)
		}
		p.Stem = strings.TrimSuffix(e.Name(), pageSufix)
		if p.Name == "" || len(p.Revisions) == 0 {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (a *Archive) sourcePath(slug, stem string) string {
	path, _ := a.find(slug, pagesDir, stem+sourceExt)
	return path
}

// FilePath is where an attachment sits. Wikidot quotes the page name into the
// directory, so a colon arrives as %3A.
func (a *Archive) FilePath(slug, pageName string, fileID int64) string {
	path, _ := a.find(slug, filesDir, quotePage(pageName), fmt.Sprint(fileID))
	return path
}

func quotePage(name string) string {
	return strings.ReplaceAll(name, ":", "%3A")
}
