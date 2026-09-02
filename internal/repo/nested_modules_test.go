package repo

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

func TestListPagesRendersAListUsersInItsBody(t *testing.T) {
	bin := os.Getenv(sidecar.EnvBinary)
	if bin == "" {
		t.Skipf("%s not set, skipping the nested module test", sidecar.EnvBinary)
	}
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)

	rend, err := sidecar.New(bin)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v, want nil", bin, err)
	}
	t.Cleanup(func() { rend.Close() })

	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}
	article, err := d.ArticleByName(ctx, "main")
	if err != nil {
		t.Fatalf("ArticleByName(main) err = %v, want nil", err)
	}
	viewer, err := d.UserByUsername(ctx, "probe-author")
	if err != nil {
		t.Fatalf("UserByUsername(probe-author) err = %v, want nil", err)
	}

	pc := page.NewContext(article, article, nil, viewer)
	r := newRenderRepo(ctx, d, users, loc, site, viewer, rend)
	body := "%%title%% by [[module ListUsers users=\".\"]]%%user_displayname%%[[/module]]"

	got, err := r.RenderModule(pc, "ListPages",
		map[string]string{"category": "probestars", "order": "name", "limit": "2"}, body)
	if err != nil {
		t.Fatalf("RenderModule(ListPages) err = %v, want nil", err)
	}
	_, rendered, ok := strings.Cut(got, `data-list-pages-page-id="main">`)
	if !ok {
		t.Fatalf("RenderModule(ListPages) = %q, want the list wrapper", got)
	}
	for _, want := range []string{"Probe Quarter", "Probe Rated", "Probe Author"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("RenderModule(ListPages) = %q, want substring %q", rendered, want)
		}
	}
	if strings.Contains(rendered, "[[module") {
		t.Errorf("RenderModule(ListPages) = %q, want no module tag left standing", rendered)
	}
}
