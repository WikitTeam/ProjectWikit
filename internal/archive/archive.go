// Package archive reads the backup a wikitCLI run leaves behind.
package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
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
	roots     []string
	extracted string
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

// Open reads a .tar.gz, or a directory holding unpacked sites, tarballs, or
// both. A backup that arrives as several tarballs is read by putting them in one
// directory. The archive owns the temporary directory it unpacks into, so
// callers must Close it.
func Open(path string) (*Archive, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	a := &Archive{}
	tarballs := []string{path}
	if info.IsDir() {
		a.roots = append(a.roots, path)
		tarballs, err = tarballsIn(path)
		if err != nil {
			return nil, err
		}
	}
	if len(tarballs) == 0 {
		return a, nil
	}

	dir, err := os.MkdirTemp("", "pwikit-archive-")
	if err != nil {
		return nil, err
	}
	for _, ball := range tarballs {
		if err := untar(ball, dir); err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
	}
	a.roots = append(a.roots, dir)
	a.extracted = dir
	return a, nil
}

func tarballsIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if e.IsDir() || !(strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")) {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}

// find answers where a path lives, searching the roots in the order they were
// added so an unpacked directory wins over a tarball of the same site.
func (a *Archive) find(parts ...string) (string, bool) {
	for _, root := range a.roots {
		full := filepath.Join(append([]string{root}, parts...)...)
		if _, err := os.Stat(full); err == nil {
			return full, true
		}
	}
	return "", false
}

func (a *Archive) Close() error {
	if a.extracted == "" {
		return nil
	}
	return os.RemoveAll(a.extracted)
}

// Sites lists the slugs the archive carries. A backup of one site holds one.
func (a *Archive) Sites() ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, root := range a.roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || e.Name() == usersDir || seen[e.Name()] {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, e.Name(), metaDir, siteFile)); err != nil {
				continue
			}
			seen[e.Name()] = true
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
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

// Users merges every _users directory the archive carries, the shared one and
// the one a site brought along. Where two disagree the newer fetch wins.
func (a *Archive) Users() (map[int64]User, error) {
	slugs, err := a.Sites()
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, root := range a.roots {
		dirs = append(dirs, filepath.Join(root, usersDir))
		for _, slug := range slugs {
			dirs = append(dirs, filepath.Join(root, slug, usersDir))
		}
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

func untar(from, into string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read %q as gzip: %w", from, err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		head, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(into, head.Name)
		if err != nil {
			return err
		}
		switch head.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, reader); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

// An entry naming its way out of the directory is refused rather than cleaned,
// because a backup that carries one is not a backup this reader understands.
func safeJoin(root, name string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q leaves the directory", name)
	}
	return target, nil
}
