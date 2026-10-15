package modules

import (
	"regexp"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/thumb"
)

func init() {
	module.Register("gallery", renderGallery)
	module.Register("gallerylist", renderGalleryList)
}

const defaultGallerySize = "medium"

type galleryItem struct {
	href, src, alt, target string
	lightbox               bool
}

func gallerySize(env module.Env, params map[string]string) (thumb.Size, error) {
	name := strings.TrimSpace(params["size"])
	if name == "" {
		name = defaultGallerySize
	}
	size, ok := thumb.Lookup(name)
	if !ok {
		return thumb.Size{}, &module.Error{Message: env.Text("module-gallery-size", "size", params["size"])}
	}
	return size, nil
}

func renderGallery(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Page == nil || env.Page.Article == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}
	size, err := gallerySize(env, params)
	if err != nil {
		return "", err
	}
	files, err := env.Data.ArticleFiles(env.Page.Article.ID)
	if err != nil {
		return "", err
	}

	viewer := module.BoolParam(params, "viewer", true)
	page := env.Page.Article.FullName()
	var items []galleryItem
	for _, f := range files {
		if !strings.HasPrefix(f.MimeType, "image/") {
			continue
		}
		items = append(items, galleryItem{
			href:     "/local--files/" + page + "/" + f.Name,
			src:      resizedURL(page, f.Name, size),
			lightbox: viewer,
		})
	}
	return galleryBox(items, size), nil
}

var galleryAttr = regexp.MustCompile(`([A-Za-z_-]+)\s*=\s*"([^"]*)"`)

func renderGalleryList(env module.Env, params map[string]string, body string) (string, error) {
	size, err := gallerySize(env, params)
	if err != nil {
		return "", err
	}
	page := ""
	if env.Page != nil && env.Page.Article != nil {
		page = env.Page.Article.FullName()
	}
	viewer := module.BoolParam(params, "viewer", true)

	var items []galleryItem
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, " ", " "))
		rest, ok := strings.CutPrefix(line, ":")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		source, attrs, _ := strings.Cut(rest, " ")
		if source == "" {
			continue
		}
		item, ok := gallerySource(source, page, size)
		if !ok {
			continue
		}
		item.lightbox = viewer
		for _, m := range galleryAttr.FindAllStringSubmatch(attrs, -1) {
			switch strings.ToLower(m[1]) {
			case "alt":
				item.alt = m[2]
			case "link":
				link := strings.TrimSpace(m[2])
				if target, ok := strings.CutPrefix(link, "*"); ok {
					link, item.target = target, "_blank"
				}
				if link != "" {
					item.href, item.lightbox = galleryLink(link), false
				}
			}
		}
		items = append(items, item)
	}
	return galleryBox(items, size), nil
}

func gallerySource(source, page string, size thumb.Size) (galleryItem, bool) {
	if isGalleryURL(source) {
		return galleryItem{href: source, src: source}, true
	}
	owner, name := page, source
	if i := strings.LastIndex(source, "/"); i >= 0 {
		owner, name = strings.TrimPrefix(source[:i], "/"), source[i+1:]
	}
	if owner == "" || name == "" {
		return galleryItem{}, false
	}
	return galleryItem{href: "/local--files/" + owner + "/" + name, src: resizedURL(owner, name, size)}, true
}

func isGalleryURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//")
}

func galleryLink(link string) string {
	if isGalleryURL(link) || strings.HasPrefix(link, "/") || strings.HasPrefix(link, "#") {
		return link
	}
	return "/" + link
}

func resizedURL(page, name string, size thumb.Size) string {
	return "/local--resized-images/" + page + "/" + name + "/" + size.Name + ".jpg"
}

func galleryBox(items []galleryItem, size thumb.Size) string {
	var b strings.Builder
	b.WriteString(`<div class="gallery-box">`)
	for _, item := range items {
		b.WriteString("\n" + ind8 + `<div class="gallery-item ` + escape.HTML(size.Name) + `">` +
			`<table><tr><td><a href="` + escape.HTML(item.href) + `"`)
		if item.lightbox {
			b.WriteString(` class="with-lb"`)
		}
		if item.target != "" {
			b.WriteString(` target="` + escape.HTML(item.target) + `"`)
		}
		b.WriteString(`><img src="` + escape.HTML(item.src) + `" alt="` + escape.HTML(item.alt) +
			`" class="gallery-image-size-` + escape.HTML(size.Name) + `"></a></td></tr></table></div>`)
	}
	b.WriteString("</div>")
	return b.String()
}
