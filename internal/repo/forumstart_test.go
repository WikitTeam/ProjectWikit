package repo

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

var update = flag.Bool("update", false, "rewrite the golden file and the corpus the oracle reads")

const (
	forumStartGolden = "testdata/forumstart.golden"
	forumStartCorpus = "testdata/forumstart_corpus.json"
)

type forumStartCase struct {
	Name    string            `json:"name"`
	Viewer  string            `json:"viewer"`
	Section string            `json:"section,omitempty"`
	Path    map[string]string `json:"path,omitempty"`
}

func forumStartCases() []forumStartCase {
	return []forumStartCase{
		{Name: "anonymous", Viewer: ""},
		{Name: "anonymous-hidden-shown", Viewer: "", Path: map[string]string{"hidden": "show"}},
		{Name: "anonymous-hidden-junk", Viewer: "", Path: map[string]string{"hidden": "maybe"}},
		{Name: "anonymous-open-section", Viewer: "", Section: "Probe Open"},
		{Name: "anonymous-hidden-section", Viewer: "", Section: "Probe Hidden"},
		{Name: "anonymous-staff-section", Viewer: "", Section: "Probe Staff"},
		{Name: "anonymous-section-and-hidden", Viewer: "", Section: "Probe Open", Path: map[string]string{"hidden": "show"}},
		{Name: "anonymous-missing-section", Viewer: "", Path: map[string]string{"s": "999999"}},
		{Name: "anonymous-unparsable-section", Viewer: "", Path: map[string]string{"s": "not-a-number"}},
		{Name: "registered", Viewer: "probe-author"},
		{Name: "superuser", Viewer: "probe-staff"},
		{Name: "superuser-hidden-shown", Viewer: "probe-staff", Path: map[string]string{"hidden": "show"}},
		{Name: "superuser-staff-section", Viewer: "probe-staff", Section: "Probe Staff"},
	}
}

func TestForumStartMatchesGolden(t *testing.T) {
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := forumStartCases()

	article, err := d.ArticleByName(ctx, "forum:start")
	if err != nil {
		t.Fatalf("ArticleByName(forum:start) err = %v, want nil", err)
	}
	sections, err := d.ForumSections(ctx)
	if err != nil {
		t.Fatalf("ForumSections() err = %v, want nil", err)
	}

	var b strings.Builder
	for _, c := range cases {
		var viewer *db.User
		if c.Viewer != "" {
			viewer, err = d.UserByUsername(ctx, c.Viewer)
			if err != nil {
				t.Fatalf("UserByUsername(%q) err = %v, want nil", c.Viewer, err)
			}
		}

		path := forumStartPath(t, c, sections)
		pc := page.NewContext(article, article, path, viewer)
		r := New(ctx, d, users, Options{Loc: loc, User: viewer})

		body, err := r.RenderModule(pc, "ForumStart", nil, "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\ntitle: %s\nstatus: %d\n%s\n", c.Name, pc.Title, pc.Status, body)
	}
	compareGolden(t, b.String(), cases, forumStartGolden, forumStartCorpus)
}

func forumStartPath(t *testing.T, c forumStartCase, sections []db.ForumSection) page.PathParams {
	t.Helper()
	path := sortedPath(c.Path)
	if c.Section == "" {
		return path
	}
	for _, section := range sections {
		if section.Name == c.Section {
			return path.Put(page.PathParam{Key: "s", Value: strconv.FormatInt(section.ID, 10)})
		}
	}
	t.Fatalf("ForumSections() has no section named %q, want one seeded", c.Section)
	return nil
}

func sortedPath(params map[string]string) page.PathParams {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var path page.PathParams
	for _, key := range keys {
		path = path.Put(page.PathParam{Key: key, Value: params[key]})
	}
	return path
}

func compareGolden(t *testing.T, got string, corpus any, goldenPath, corpusPath string) {
	t.Helper()
	if *update {
		data, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			t.Fatalf("Marshal(%s) err = %v, want nil", corpusPath, err)
		}
		if err := os.WriteFile(filepath.FromSlash(corpusPath), append(data, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", corpusPath, err)
		}
		if err := os.WriteFile(filepath.FromSlash(goldenPath), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", goldenPath, err)
		}
		return
	}
	want, err := os.ReadFile(filepath.FromSlash(goldenPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", goldenPath, err)
	}
	if got != string(want) {
		gotAt, wantAt := firstDiff(got, string(want))
		t.Errorf("%s = %q, want %q", goldenPath, gotAt, wantAt)
	}
}

func firstDiff(got, want string) (string, string) {
	at := 0
	for at < len(got) && at < len(want) && got[at] == want[at] {
		at++
	}
	cut := func(s string) string { return s[max(0, at-40):min(len(s), at+40)] }
	return cut(got), cut(want)
}
