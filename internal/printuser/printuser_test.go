package printuser

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

var update = flag.Bool("update", false, "rewrite the golden file and the corpus the oracle reads")

const (
	goldenPath = "testdata/printuser.golden"
	corpusPath = "testdata/render_corpus.json"
)

func testRenderer(t *testing.T, c corpusFile) *Renderer {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	load := func(name string) (string, error) {
		svg, ok := c.Icons[name]
		if !ok {
			return "", fmt.Errorf("no icon %q", name)
		}
		return svg, nil
	}
	return New(bundle.Localizer(i18n.DefaultLanguage), load)
}

func categoryIDs(specs []roleSpec) map[string]int64 {
	ids := make(map[string]int64)
	for _, spec := range specs {
		if spec.Category == "" {
			continue
		}
		if _, ok := ids[spec.Category]; !ok {
			ids[spec.Category] = int64(len(ids) + 1)
		}
	}
	return ids
}

func buildRoles(specs []roleSpec) map[string]roles.Role {
	ids := categoryIDs(specs)
	out := make(map[string]roles.Role, len(specs))
	for i, spec := range specs {
		role := roles.Role{
			ID:                int64(i + 1),
			Slug:              spec.Slug,
			Name:              spec.Name,
			ShortName:         spec.ShortName,
			Index:             spec.Index,
			InlineVisualMode:  roles.InlineVisualMode(spec.InlineVisualMode),
			ProfileVisualMode: roles.ProfileVisualMode(spec.ProfileVisualMode),
			Color:             spec.Color,
			Icon:              spec.Icon,
			BadgeText:         spec.BadgeText,
			BadgeBg:           spec.BadgeBg,
			BadgeTextColor:    spec.BadgeTextColor,
			BadgeShowBorder:   spec.BadgeShowBorder,
		}
		if spec.Category != "" {
			id := ids[spec.Category]
			role.CategoryID = &id
		}
		out[spec.Slug] = role
	}
	return out
}

// selected returns the roles of one case in index order, which is the order
// RolesByUser hands them over in.
func selected(all map[string]roles.Role, slugs []string) []roles.Role {
	var out []roles.Role
	for _, role := range all {
		for _, slug := range slugs {
			if role.Slug == slug {
				out = append(out, role)
			}
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Index < out[j-1].Index; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func render(t *testing.T, r *Renderer, all map[string]roles.Role, c caseSpec) string {
	t.Helper()
	opts := Options{Avatar: c.Avatar, Hover: c.Hover}
	switch c.Kind {
	case "system":
		return r.System(opts)
	case "anonymous":
		return r.Anonymous(opts)
	case "external":
		return r.External(c.External, opts)
	case "user":
		html, err := r.User(User{
			ID:              c.User.ID,
			Type:            c.User.Type,
			Username:        c.User.Username,
			WikidotUsername: c.User.WikidotUsername,
			DisplayName:     c.User.DisplayName,
			Avatar:          c.User.Avatar,
			IsActive:        c.User.IsActive,
		}, selected(all, c.Roles), opts)
		if err != nil {
			t.Fatalf("User(%s) err = %v, want nil", c.Name, err)
		}
		return html
	}
	t.Fatalf("case %s has kind %q, want one of system, anonymous, external, user", c.Name, c.Kind)
	return ""
}

func TestRenderMatchesGolden(t *testing.T) {
	c := corpus()
	r := testRenderer(t, c)
	all := buildRoles(c.Roles)

	var b strings.Builder
	for _, spec := range c.Cases {
		fmt.Fprintf(&b, "=== %s\n%s\n", spec.Name, render(t, r, all, spec))
	}
	got := b.String()

	if *update {
		writeCorpus(t, c)
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
		t.Errorf("render = %q, want %q", gotAt, wantAt)
	}
}

func firstDiff(got, want string) (string, string) {
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			return excerpt(got, i), excerpt(want, i)
		}
	}
	return excerpt(got, min(len(got), len(want))), excerpt(want, min(len(got), len(want)))
}

func excerpt(s string, at int) string {
	start := max(0, at-40)
	end := min(len(s), at+40)
	return s[start:end]
}

func writeCorpus(t *testing.T, c corpusFile) {
	t.Helper()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(corpus) err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.FromSlash(corpusPath), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", corpusPath, err)
	}
}

func TestTailsPrefersBannedOverBot(t *testing.T) {
	r := testRenderer(t, corpus())
	tails, err := r.Tails(User{Type: TypeBot, IsActive: false}, nil)
	if err != nil {
		t.Fatalf("Tails() err = %v, want nil", err)
	}
	if len(tails.Badges) != 1 {
		t.Fatalf("len(Tails().Badges) = %d, want 1", len(tails.Badges))
	}
	if tails.Badges[0].Bg != "#000000" {
		t.Errorf("Tails().Badges[0].Bg = %q, want %q", tails.Badges[0].Bg, "#000000")
	}
}

func TestUserRendersWithoutAnIconThatWillNotLoad(t *testing.T) {
	r := New(nil, func(string) (string, error) { return "", errors.New("no such icon") })
	role := roles.Role{Slug: "icon", InlineVisualMode: roles.InlineIcon, Icon: "-/roles/gone.svg"}

	got, err := r.User(User{ID: 1, Type: TypeNormal, Username: "u", IsActive: true}, []roles.Role{role}, Options{Avatar: true})
	if err != nil {
		t.Fatalf("User() err = %v, want nil", err)
	}
	if !strings.Contains(got, `data-user-name="u"`) {
		t.Errorf("User() = %q, want the chip anyway", got)
	}
	if strings.Contains(got, "role-icon") {
		t.Errorf("User() = %q, want no icon in it", got)
	}
}

func TestURLName(t *testing.T) {
	cases := []struct {
		user User
		want string
	}{
		{User{Type: TypeNormal, Username: "local", WikidotUsername: "remote"}, "local"},
		{User{Type: TypeWikidot, Username: "local", WikidotUsername: "remote"}, "remote"},
		{User{Type: TypeWikidot, Username: "local"}, "local"},
	}
	for _, c := range cases {
		if got := c.user.URLName(); got != c.want {
			t.Errorf("URLName(%+v) = %q, want %q", c.user, got, c.want)
		}
	}
}
