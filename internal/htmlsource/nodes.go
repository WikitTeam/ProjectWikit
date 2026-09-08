package htmlsource

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func attr(n *html.Node, name string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// attrs writes every attribute back in the order it was parsed, which is the
// order it was written, so a round trip does not reshuffle a tag.
func attrs(n *html.Node, skip ...string) string {
	var b strings.Builder
	for _, a := range n.Attr {
		if contains(skip, a.Key) {
			continue
		}
		b.WriteString(" " + a.Key + `="` + escapeValue(a.Val) + `"`)
	}
	return b.String()
}

func escapeValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	return strings.ReplaceAll(v, `"`, `\"`)
}

func classes(n *html.Node) []string {
	if n == nil {
		return nil
	}
	return strings.Fields(attr(n, "class"))
}

func hasAnyClass(n *html.Node) bool { return len(classes(n)) > 0 }

func hasClass(n *html.Node, want ...string) bool {
	held := classes(n)
	for _, one := range want {
		if contains(held, one) {
			return true
		}
	}
	return false
}

func contains(list []string, want string) bool {
	for _, one := range list {
		if one == want {
			return true
		}
	}
	return false
}

func find(n *html.Node, tag, class string) *html.Node {
	if n == nil {
		return nil
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if matches(child, tag, class) {
			return child
		}
		if found := find(child, tag, class); found != nil {
			return found
		}
	}
	return nil
}

func findAll(n *html.Node, tag, class string) []*html.Node {
	if n == nil {
		return nil
	}
	var out []*html.Node
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if matches(child, tag, class) {
			out = append(out, child)
			continue
		}
		out = append(out, findAll(child, tag, class)...)
	}
	return out
}

func matches(n *html.Node, tag, class string) bool {
	if n.Type != html.ElementNode || n.Data != tag {
		return false
	}
	return class == "" || hasClass(n, class)
}

func text(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return b.String()
}

func oneLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}

func spaced(s string) string { return strings.ReplaceAll(s, "\n", " ") }

func itoa(n int) string { return strconv.Itoa(n) }

func userName(n *html.Node) string {
	href := attr(find(n, "a", ""), "href")
	if cut := strings.LastIndex(href, "/"); cut >= 0 {
		return href[cut+1:]
	}
	return href
}

// Every reference in the document reaches for the same block at the end of it,
// so the bodies are collected once.
func (c *converter) footnotesFrom(root *html.Node) {
	block := root
	if !matches(root, "div", "footnotes-footer") {
		block = find(root, "div", "footnotes-footer")
	}
	if block == nil {
		return
	}
	if c.footnotes == nil {
		c.footnotes = map[string]string{}
	}
	for _, note := range findAll(block, "div", "footnote-footer") {
		link := find(note, "a", "")
		number := strings.TrimSpace(text(link))
		if link != nil && link.Parent != nil {
			link.Parent.RemoveChild(link)
		}
		c.footnotes[number] = strings.TrimSpace(cutLabel(c.children(note)))
	}
}
