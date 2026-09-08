package archive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bodgit/sevenzip"
)

const (
	forumDir     = "forum"
	categoryDir  = "category"
	postBodyFile = "latest.html"
)

type Category struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Thread struct {
	ID          int64  `json:"id"`
	CategoryID  int64  `json:"-"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Started     int64  `json:"started"`
	StartedUser *int64 `json:"startedUser"`
	Last        int64  `json:"last"`
	Sticky      bool   `json:"sticky"`
	Locked      bool   `json:"isLocked"`
	Posts       []Post `json:"posts"`
}

type Post struct {
	ID        int64          `json:"id"`
	Title     string         `json:"title"`
	Poster    int64          `json:"poster"`
	Stamp     int64          `json:"stamp"`
	Revisions []PostRevision `json:"revisions"`
	Children  []Post         `json:"children"`
}

type PostRevision struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Author int64  `json:"author"`
	Stamp  int64  `json:"stamp"`
}

func (a *Archive) Categories(slug string) ([]Category, error) {
	dir, ok := a.find(slug, metaDir, forumDir, categoryDir)
	if !ok {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Category
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), pageSufix) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var c Category
		if err := json.Unmarshal(body, &c); err != nil {
			return nil, fmt.Errorf("read the forum category %q: %w", e.Name(), err)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (a *Archive) Threads(slug string, categoryID int64) ([]Thread, error) {
	dir, ok := a.find(slug, metaDir, forumDir, strconv.FormatInt(categoryID, 10))
	if !ok {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Thread
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), pageSufix) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var t Thread
		if err := json.Unmarshal(body, &t); err != nil {
			return nil, fmt.Errorf("read the forum thread %q: %w", e.Name(), err)
		}
		t.CategoryID = categoryID
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started < out[j].Started })
	return out, nil
}

func (a *Archive) PostBodies(slug string, thread Thread) (map[string]string, error) {
	out := map[string]string{}
	path, ok := a.find(slug, forumDir, strconv.FormatInt(thread.CategoryID, 10),
		strconv.FormatInt(thread.ID, 10)+sourceExt)
	if !ok {
		return out, nil
	}
	reader, err := sevenzip.OpenReader(path)
	if err != nil {
		return out, nil
	}
	defer reader.Close()

	for _, entry := range reader.File {
		body, err := readEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("read %s of thread %d: %w", entry.Name, thread.ID, err)
		}
		out[filepath.ToSlash(entry.Name)] = body
	}
	return out, nil
}
