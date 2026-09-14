// Package htmlsource turns the forum HTML a backup carries back into wikitext.
package htmlsource

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Convert reads a fragment, not a document, because a forum post is a piece of
// a page and never carries its own html element.
func Convert(source string) string {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(source), body)
	if err != nil {
		return ""
	}
	c := &converter{}
	for _, n := range nodes {
		c.footnotesFrom(n)
	}
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(c.node(n))
	}
	return b.String()
}

type converter struct {
	footnotes map[string]string
}

func (c *converter) children(n *html.Node) string {
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(c.node(child))
	}
	return b.String()
}

func (c *converter) node(n *html.Node) string {
	switch n.Type {
	case html.TextNode:
		// A newline inside the HTML is layout, and wikitext reads it as a break.
		return strings.ReplaceAll(n.Data, "\n", "")
	case html.ElementNode:
	default:
		return ""
	}

	switch n.Data {
	case "p":
		return c.children(n) + "\n\n"
	case "em":
		return "//" + c.children(n) + "//"
	case "strong", "b":
		return "**" + c.children(n) + "**"
	case "u":
		return "__" + c.children(n) + "__"
	case "strike", "s":
		return "--" + c.children(n) + "--"
	case "sup":
		return c.superscript(n)
	case "sub":
		return ",," + c.children(n) + ",,"
	case "br":
		return "\n"
	case "iframe":
		return "[[iframe " + attr(n, "src") + attrs(n, "src") + "]]"
	case "span":
		return c.span(n)
	case "blockquote":
		lines := strings.Split(strings.TrimSpace(c.children(n)), "\n")
		return "> " + strings.Join(lines, "\n> ") + "\n"
	case "div":
		return c.div(n)
	case "a":
		return "[[a" + attrs(n) + "]]" + c.children(n) + "[[/a]]"
	case "img":
		return "[[image " + attr(n, "src") + attrs(n, "src", "alt") + "]]"
	case "hr":
		return "----\n"
	case "ul", "li", "ol":
		return "[[" + n.Data + "]]\n" + c.children(n) + "[[/" + n.Data + "]]\n"
	case "h1", "h2", "h3", "h4", "h5", "h6", "h7":
		return strings.Repeat("+", int(n.Data[1]-'0')) + " " + oneLine(text(n)) + "\n"
	case "tt":
		return "{{" + c.children(n) + "}}"
	case "table":
		return "[[table" + attrs(n) + "]]\n" + c.children(n) + "[[/table]]\n"
	case "tbody":
		return c.children(n)
	case "tr":
		return "[[row" + attrs(n) + "]]\n" + c.children(n) + "[[/row]]\n"
	case "td":
		return "[[cell" + attrs(n) + "]]\n" + c.children(n) + "[[/cell]]\n"
	case "th":
		return "[[hcell" + attrs(n) + "]]\n" + c.children(n) + "[[/hcell]]\n"
	case "script":
		return ""
	case "dl":
		return definitions(n)
	}
	return c.children(n)
}

func (c *converter) superscript(n *html.Node) string {
	if !hasClass(n, "footnoteref") {
		return "^^" + c.children(n) + "^^"
	}
	return "[[footnote]]" + c.footnotes[strings.TrimSpace(text(n))] + "[[/footnote]]"
}

func (c *converter) span(n *html.Node) string {
	switch {
	case hasClass(n, "printuser"):
		star := ""
		if hasClass(n, "avatarhover") {
			star = "*"
		}
		return "[[" + star + "user " + userName(n) + "]]"
	case hasClass(n, "math-inline"):
		return "[[$ " + strings.TrimSpace(strings.Trim(text(n), "$")) + " $]]"
	case hasClass(n, "equation-number"):
		return ""
	}
	return "[[span" + attrs(n) + "]]" + c.children(n) + "[[/span]]"
}

var plainDivClasses = []string{
	"rimg", "limg", "cimg", "blockquote", "сimg", "scpnet-progress-bar",
	"scpnet-progress-bar__tick", "block-error", "collapsible-block-unfolded-link",
}

func (c *converter) div(n *html.Node) string {
	switch {
	case !hasAnyClass(n) || hasClass(n, plainDivClasses...):
		return c.plainDiv(n)
	case hasClass(n, "collapsible-block"):
		return c.collapsible(n)
	case hasClass(n, "yui-navset"):
		return c.tabview(n)
	case hasClass(n, "code"):
		return c.code(n)
	case hasClass(n, "footnotes-footer"):
		return "[[footnoteblock title=\"" + escapeValue(oneLine(text(find(n, "div", "title")))) + "\"]]\n"
	case hasClass(n, "bibitems"):
		return c.bibliography(n)
	case hasClass(n, "image-container"):
		return c.image(n)
	case hasClass(n, "content-separator"):
		return "====\n"
	case hasClass(n, "math-equation"):
		return "[[math]]\n" + strings.TrimSpace(text(n)) + "\n[[/math]]\n"
	case hasClass(n, "wiki-note"):
		return "[[note]]\n" + c.children(n) + "[[/note]]\n"
	}
	return c.plainDiv(n)
}

func (c *converter) plainDiv(n *html.Node) string {
	return "[[div" + attrs(n) + "]]\n" + c.children(n) + "[[/div]]\n"
}

func (c *converter) collapsible(n *html.Node) string {
	show := oneLine(text(find(find(n, "div", "collapsible-block-folded"), "a", "collapsible-block-link")))
	hide := oneLine(text(find(find(find(n, "div", "collapsible-block-unfolded"),
		"div", "collapsible-block-unfolded-link"), "a", "collapsible-block-link")))
	return "[[collapsible show=\"" + escapeValue(show) + "\" hide=\"" + escapeValue(hide) + "\"]]\n" +
		c.children(find(n, "div", "collapsible-block-content")) + "[[/collapsible]]\n"
}

func (c *converter) tabview(n *html.Node) string {
	var titles []string
	for _, li := range findAll(find(n, "ul", "yui-nav"), "li", "") {
		titles = append(titles, oneLine(text(li)))
	}
	var b strings.Builder
	b.WriteString("[[tabview]]\n")
	for i, tab := range findAll(find(n, "div", "yui-content"), "div", "") {
		title := ""
		if i < len(titles) {
			title = titles[i]
		}
		b.WriteString("[[tab title=\"" + escapeValue(title) + "\"]]\n")
		b.WriteString(c.children(tab))
		b.WriteString("[[/tab]]\n")
	}
	b.WriteString("[[/tabview]]\n")
	return b.String()
}

func (c *converter) code(n *html.Node) string {
	if pre := find(n, "pre", ""); pre != nil {
		if code := find(pre, "code", ""); code != nil {
			return "[[code]]\n" + text(code) + "\n[[/code]]\n"
		}
	}
	if main := find(n, "div", "hl-main"); main != nil {
		if pre := find(main, "pre", ""); pre != nil {
			return "[[code]]\n" + text(pre) + "\n[[/code]]\n"
		}
	}
	return c.plainDiv(n)
}

func (c *converter) bibliography(n *html.Node) string {
	var b strings.Builder
	b.WriteString("[[bibliography title=\"" + escapeValue(oneLine(text(find(n, "div", "title")))) + "\"]]\n")
	for i, item := range findAll(n, "div", "bibitem") {
		b.WriteString(": cite" + itoa(i+1) + " : " + strings.TrimSpace(cutLabel(c.children(item))) + "\n")
	}
	b.WriteString("[[/bibliography]]\n")
	return b.String()
}

var imagePrefixes = [][2]string{
	{"floatleft", "f<"}, {"floatright", "f>"},
	{"alignleft", "<"}, {"alignright", ">"}, {"aligncenter", "="},
}

func (c *converter) image(n *html.Node) string {
	prefix := ""
	for _, pair := range imagePrefixes {
		if hasClass(n, pair[0]) {
			prefix = pair[1]
		}
	}
	img := find(n, "img", "")
	if img == nil {
		return c.plainDiv(n)
	}
	return "[[" + prefix + "image " + attr(img, "src") + attrs(img, "src", "alt") + "]]"
}

func definitions(n *html.Node) string {
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "dt" {
			continue
		}
		value := ""
		for after := child.NextSibling; after != nil; after = after.NextSibling {
			if after.Type == html.ElementNode && after.Data == "dd" {
				value = text(after)
				break
			}
		}
		b.WriteString(": " + spaced(text(child)) + " : " + spaced(value) + "\n")
	}
	return b.String()
}

// The number a bibliography item opens with belongs to the list rather than to
// the text, so the first two characters go.
func cutLabel(s string) string {
	if len(s) > 2 {
		return s[2:]
	}
	return s
}
