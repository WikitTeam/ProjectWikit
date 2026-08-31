package listpages

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type Sections struct {
	Head string
	Body string
	Foot string
}

// The two characters this eats around each part are not a mistake to fix. They
// are what the running site's output already has, so removing them would move
// every listing by two characters.
var sectionPattern = regexp.MustCompile(`(?is)\A(?:.*\s*(\[\[head]]\n?.(?P<head>.*?).\[\[/head]])|)` +
	`(?:.*\s*(\[\[body]]\n?.(?P<body>.*?).\[\[/body]])|)` +
	`(?:.*\s*(\[\[foot]]\n?.(?P<foot>.*?).\[\[/foot]])|)`)

func Split(content string) Sections {
	match := sectionPattern.FindStringSubmatch(content)
	if match == nil {
		return Sections{}
	}
	var out Sections
	for i, name := range sectionPattern.SubexpNames() {
		switch name {
		case "head":
			out.Head = match[i]
		case "body":
			out.Body = match[i]
		case "foot":
			out.Foot = match[i]
		}
	}
	return out
}

const urlParamPrefix = "@url|"

func URLParams(params map[string]string, path page.PathParams) (values map[string]string, null map[string]bool) {
	null = map[string]bool{}
	for key, value := range params {
		if len(value) < len(urlParamPrefix) ||
			!strings.EqualFold(value[:len(urlParamPrefix)], urlParamPrefix) {
			continue
		}
		if param, ok := path.Lookup(key); ok {
			params[key] = param.Value
			if param.Bare {
				null[key] = true
			}
			continue
		}
		params[key] = value[len(urlParamPrefix):]
	}
	return params, null
}

func BasePath(fullName string, path page.PathParams) string {
	if fullName == "" {
		return "#"
	}
	out := "/" + fullName
	for _, param := range path {
		if param.Key == "p" {
			continue
		}
		out += "/" + quotePlus(param.Key) + "/" + quotePlus(pathValue(param))
	}
	return out
}

func pathValue(param page.PathParam) string {
	if param.Bare {
		return "None"
	}
	return param.Value
}

func quotePlus(s string) string {
	return strings.ReplaceAll(page.QuoteAll(s), "%20", "+")
}

const (
	ind12 = "            "
	ind16 = "                "
	ind20 = "                    "
	ind24 = "                        "
)

func Pagination(loc *i18n.Localizer, basePath string, current, total int) string {
	if total <= 1 {
		return ""
	}
	const around = 2

	leftFrom := 1
	leftTo := leftFrom + 1
	if current < around*2+1 {
		leftTo = around + 1
	}
	if leftTo > total-1 {
		leftTo = total - 1
	}
	rightTo := total
	rightFrom := max(leftTo+1, rightTo-1)
	if current > rightTo-(around*2+1) {
		rightFrom = max(leftTo+1, total-(around+1))
	}
	centerFrom := max(leftTo+1, current-around)
	centerTo := min(rightFrom-1, current+around)

	var b strings.Builder
	b.WriteString(`<div class="pager">` + "\n" + ind16)
	b.WriteString(`<span class="pager-no">` +
		text(loc, "module-listpages-pager-count", "page", strconv.Itoa(current), "total", strconv.Itoa(total)) +
		`</span>` + "\n" + ind16)

	if current > 1 {
		b.WriteString("\n" + ind20 + step(loc, basePath, current-1, "module-listpages-pager-prev") + "\n" + ind16)
	}
	b.WriteString("\n" + ind16)

	writeRange := func(class string, from, to int) {
		for p := from; p <= to; p++ {
			b.WriteString("\n" + ind20 + "\n" + ind24 + number(basePath, class, p, p == current) + "\n" + ind20 + "\n" + ind16)
		}
	}
	writeDots := func(show bool) {
		if show {
			b.WriteString("\n" + ind20 + `<span class="dots">...</span>` + "\n" + ind16)
		}
		b.WriteString("\n" + ind16)
	}

	writeRange("1", leftFrom, leftTo)
	b.WriteString("\n" + ind16)
	writeDots(centerFrom > leftTo+1)
	writeRange("2", centerFrom, centerTo)
	b.WriteString("\n" + ind16)
	writeDots(centerTo < rightFrom-1)
	writeRange("3", rightFrom, rightTo)
	b.WriteString("\n" + ind16)

	if current < total {
		b.WriteString("\n" + ind20 + step(loc, basePath, current+1, "module-listpages-pager-next") + "\n" + ind16)
	}
	b.WriteString("\n" + ind12 + "</div>")
	return b.String()
}

func step(loc *i18n.Localizer, basePath string, target int, label string) string {
	return `<span class="target"><a href="` + pageHref(basePath, target) +
		`" data-pagination-target="` + strconv.Itoa(target) + `">` + text(loc, label) + `</a></span>`
}

func number(basePath, class string, p int, current bool) string {
	if current {
		return `<span class="` + class + ` target current">` + strconv.Itoa(p) + `</span>`
	}
	return `<span class="` + class + ` target"><a href="` + pageHref(basePath, p) +
		`" data-pagination-target="` + strconv.Itoa(p) + `">` + strconv.Itoa(p) + `</a></span>`
}

func pageHref(basePath string, p int) string {
	if basePath == "" {
		return "#"
	}
	return escape.HTML(basePath) + "/p/" + strconv.Itoa(p)
}

func text(loc *i18n.Localizer, id string, args ...any) string {
	if loc == nil {
		return id
	}
	return loc.T(id, args...)
}

func Wrap(content, pagination, pathParams, params, body, pageID string) string {
	return `<div class="list-pages-box w-list-pages"` + "\n" +
		ind20 + ` data-list-pages-path-params="` + escape.HTML(pathParams) + `"` + "\n" +
		ind20 + ` data-list-pages-params="` + escape.HTML(params) + `"` + "\n" +
		ind20 + ` data-list-pages-content="` + escape.HTML(body) + `"` + "\n" +
		ind20 + ` data-list-pages-page-id="` + escape.HTML(pageID) + `">` + "\n" +
		ind16 + content + "\n" +
		ind16 + pagination + "\n" +
		ind16 + "</div>"
}
