package pagerender

import (
	"context"
	"os"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func linkEnv(t *testing.T) (*Env, *db.Site) {
	t.Helper()
	dsn := os.Getenv(db.EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", db.EnvDSN)
	}
	ctx := context.Background()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	t.Cleanup(conn.Close)

	slugs, err := conn.SiteSlugs(ctx)
	if err != nil {
		t.Fatalf("SiteSlugs() err = %v, want nil", err)
	}
	if len(slugs) == 0 {
		t.Skip("the test database holds no site")
	}
	current, err := conn.SiteBySlug(ctx, slugs[0])
	if err != nil {
		t.Fatalf("SiteBySlug(%q) err = %v, want nil", slugs[0], err)
	}
	return Deps{DB: conn}.Env(ctx, nil, current, nil), current
}

func TestLinkFilesANamedSiteAgainstThatSite(t *testing.T) {
	env, current := linkEnv(t)

	got, err := env.link(":"+current.Slug+":Main", db.LinkInclude)
	if err != nil {
		t.Fatalf("link() err = %v, want nil", err)
	}
	if got.To != "main" {
		t.Errorf("link(:%s:Main).To = %q, want %q", current.Slug, got.To, "main")
	}
	if got.ToSiteID == nil || *got.ToSiteID != current.ID {
		t.Errorf("link(:%s:Main).ToSiteID = %v, want %d", current.Slug, got.ToSiteID, current.ID)
	}
}

func TestLinkLeavesAnUnknownSiteWhole(t *testing.T) {
	env, _ := linkEnv(t)

	got, err := env.link(":nosuchsite:main", db.LinkPlain)
	if err != nil {
		t.Fatalf("link() err = %v, want nil", err)
	}
	if got.To != ":nosuchsite:main" {
		t.Errorf("link(:nosuchsite:main).To = %q, want %q", got.To, ":nosuchsite:main")
	}
	if got.ToSiteID != nil {
		t.Errorf("link(:nosuchsite:main).ToSiteID = %v, want nil", got.ToSiteID)
	}
}

func TestLinkWithoutASiteStaysLocal(t *testing.T) {
	env, _ := linkEnv(t)

	got, err := env.link("Component:Box", db.LinkPlain)
	if err != nil {
		t.Fatalf("link() err = %v, want nil", err)
	}
	if got.To != "component:box" {
		t.Errorf("link(Component:Box).To = %q, want %q", got.To, "component:box")
	}
	if got.ToSiteID != nil {
		t.Errorf("link(Component:Box).ToSiteID = %v, want nil", got.ToSiteID)
	}
}
