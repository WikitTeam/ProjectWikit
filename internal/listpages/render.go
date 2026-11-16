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

var sectionPattern = regexp.MustCompile(`(?is)\A(?:.*\s*(\[\[head]]\n*(?P<head>.*?)\n*\[\[/head]])|)` +
	`(?:.*\s*(\[\[body]]\n*(?P<body>.*?)\n*\[\[/body]])|)` +
	`(?:.*\s*(\[\[foot]]\n*(?P<foot>.*?)\n*\[\[/foot]])|)`)

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

const (
	urlParam       = "@url"
	urlParamPrefix = urlParam + "|"
)

func URLParams(params map[string]string, path page.PathParams) (values map[string]string, null map[string]bool) {
	null = map[string]bool{}
	prefix := strings.TrimSpace(params["urlattrprefix"])
	for key, value := range params {
		fallback, ok := urlDefault(value)
		if !ok {
			continue
		}
		if param, ok := lookupURLParam(path, prefix+key); ok {
			params[key] = param.Value
			if param.Bare {
				null[key] = true
			}
			continue
		}
		params[key] = fallback
	}
	return params, null
}

func urlDefault(value string) (string, bool) {
	if strings.EqualFold(value, urlParam) {
		return value, true
	}
	if len(value) < len(urlParamPrefix) ||
		!strings.EqualFold(value[:len(urlParamPrefix)], urlParamPrefix) {
		return "", false
	}
	return value[len(urlParamPrefix):], true
}

func lookupURLParam(path page.PathParams, key string) (page.PathParam, bool) {
	if param, ok := path.Lookup(key); ok {
		return param, true
	}
	if base, ok := strings.CutSuffix(key, "tags"); ok {
		return path.Lookup(base + "tag")
	}
	return page.PathParam{}, false
}

func BasePath(fullName string, path page.PathParams) string {
	return KeyedBasePath(fullName, path, "p")
}

func KeyedBasePath(fullName string, path page.PathParams, pageKey string) string {
	if fullName == "" {
		return "#"
	}
	out := "/" + fullName
	for _, param := range path {
		if param.Key == pageKey {
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
	return KeyedPagination(loc, basePath, "p", current, total)
}

func KeyedPagination(loc *i18n.Localizer, basePath, pageKey string, current, total int) string {
	return pagination(loc, func(p int) string { return pageHref(basePath, pageKey, p) }, true, current, total)
}

// PaginationLinks pages a view reached by its own URL rather than swapped in
// place, so the numbers carry no hook for the script that re-renders a module.
func PaginationLinks(loc *i18n.Localizer, href func(page int) string, current, total int) string {
	return pagination(loc, href, false, current, total)
}

func pagination(loc *i18n.Localizer, href func(int) string, hook bool, current, total int) string {
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
		b.WriteString("\n" + ind20 + step(loc, href, hook, current-1, "module-listpages-pager-prev") + "\n" + ind16)
	}
	b.WriteString("\n" + ind16)

	writeRange := func(class string, from, to int) {
		for p := from; p <= to; p++ {
			b.WriteString("\n" + ind20 + "\n" + ind24 + number(href, hook, class, p, p == current) + "\n" + ind20 + "\n" + ind16)
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
		b.WriteString("\n" + ind20 + step(loc, href, hook, current+1, "module-listpages-pager-next") + "\n" + ind16)
	}
	b.WriteString("\n" + ind12 + "</div>")
	return b.String()
}

func step(loc *i18n.Localizer, href func(int) string, hook bool, target int, label string) string {
	return `<span class="target"><a href="` + href(target) + hookAttr(hook, target) +
		`>` + text(loc, label) + `</a></span>`
}

func number(href func(int) string, hook bool, class string, p int, current bool) string {
	if current {
		return `<span class="` + class + ` target current">` + strconv.Itoa(p) + `</span>`
	}
	return `<span class="` + class + ` target"><a href="` + href(p) + hookAttr(hook, p) +
		`>` + strconv.Itoa(p) + `</a></span>`
}

func hookAttr(hook bool, target int) string {
	if !hook {
		return `"`
	}
	return `" data-pagination-target="` + strconv.Itoa(target) + `"`
}

func pageHref(basePath, pageKey string, p int) string {
	if basePath == "" {
		return "#"
	}
	return escape.HTML(basePath) + "/" + escape.HTML(quotePlus(pageKey)) + "/" + strconv.Itoa(p)
}

func text(loc *i18n.Localizer, id string, args ...any) string {
	if loc == nil {
		return id
	}
	return loc.T(id, args...)
}

func Wrap(content, pagination, pathParams, params, body, pageID string) string {
	return KeyedWrap(content, pagination, pathParams, params, body, pageID, "p")
}

// The page key is written out only when it is not the usual one, so a listing
// without urlAttrPrefix comes out as it always has.
func KeyedWrap(content, pagination, pathParams, params, body, pageID, pageKey string) string {
	key := ""
	if pageKey != "p" {
		key = ind20 + ` data-list-pages-page-key="` + escape.HTML(pageKey) + `"` + "\n"
	}
	return `<div class="list-pages-box w-list-pages"` + "\n" + key +
		ind20 + ` data-list-pages-path-params="` + escape.HTML(pathParams) + `"` + "\n" +
		ind20 + ` data-list-pages-params="` + escape.HTML(params) + `"` + "\n" +
		ind20 + ` data-list-pages-content="` + escape.HTML(body) + `"` + "\n" +
		ind20 + ` data-list-pages-page-id="` + escape.HTML(pageID) + `">` + "\n" +
		ind16 + content + "\n" +
		ind16 + pagination + "\n" +
		ind16 + "</div>"
}
