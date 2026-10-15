package modules

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func galleryEnv() module.Env {
	article := &db.Article{Category: "_default", Name: "here"}
	return module.Env{Page: page.NewContext(article, article, nil, nil)}
}

func TestRenderGalleryListSources(t *testing.T) {
	body := "\n: https://example.org/a.png\n: shot.jpg alt=\"A shot\"\n: other/b.png link=\"*target-page\"\nnot an item\n"
	got, err := renderGalleryList(galleryEnv(), map[string]string{"size": "small"}, body)
	if err != nil {
		t.Fatalf("renderGalleryList() err = %v, want nil", err)
	}
	for _, want := range []string{
		`<a href="https://example.org/a.png" class="with-lb"><img src="https://example.org/a.png" alt="" class="gallery-image-size-small">`,
		`<a href="/local--files/here/shot.jpg" class="with-lb"><img src="/local--resized-images/here/shot.jpg/small.jpg" alt="A shot"`,
		`<a href="/target-page" target="_blank"><img src="/local--resized-images/other/b.png/small.jpg"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderGalleryList() = %q, want substring %q", got, want)
		}
	}
	if n := strings.Count(got, "gallery-item small"); n != 3 {
		t.Errorf("gallery items = %d, want 3", n)
	}
}

func TestRenderGalleryListViewerOff(t *testing.T) {
	got, _ := renderGalleryList(galleryEnv(), map[string]string{"viewer": "no"}, ": https://example.org/a.png")
	if strings.Contains(got, "with-lb") {
		t.Errorf("renderGalleryList(viewer=no) = %q, want no with-lb", got)
	}
}
