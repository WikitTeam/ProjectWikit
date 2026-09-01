package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/thumb"
)

func init() { module.Register("gallery", renderGallery) }

const defaultGallerySize = "medium"

func renderGallery(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Page == nil || env.Page.Article == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}
	name := strings.TrimSpace(params["size"])
	if name == "" {
		name = defaultGallerySize
	}
	size, ok := thumb.Lookup(name)
	if !ok {
		return "", &module.Error{Message: env.Text("module-gallery-size", "size", params["size"])}
	}

	files, err := env.Data.ArticleFiles(env.Page.Article.ID)
	if err != nil {
		return "", err
	}

	base := "/local--resized-images/" + env.Page.Article.FullName() + "/"
	var b strings.Builder
	b.WriteString(`<div class="gallery-box">`)
	for _, f := range files {
		if !strings.HasPrefix(f.MimeType, "image/") {
			continue
		}
		url := escape.HTML(base + f.Name + "/" + size.Name + ".jpg")
		b.WriteString("\n" + ind8 + `<div class="gallery-item ` + escape.HTML(size.Name) + `">` +
			`<table><tr><td><a href="` + url + `"><img src="` + url + `" alt=""></a></td></tr></table></div>`)
	}
	b.WriteString("</div>")
	return b.String(), nil
}
