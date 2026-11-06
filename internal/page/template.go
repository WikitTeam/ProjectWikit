// Package page holds the state of one page render and the substitutions that
// run before ftml sees the source.
package page

import (
	"regexp"
	"strconv"
	"strings"
)

// Go's dot excludes newlines by default, so a %% pair may not span lines.
var variablePattern = regexp.MustCompile(`%%(.*?)%%`)

// ApplyTemplate replaces every %%name%% the resolver knows. An unresolved name
// is put back verbatim rather than dropped, which is what lets a page carrying
// %% in its prose survive the pass.
func ApplyTemplate(template string, resolve func(name string) (string, bool)) string {
	return variablePattern.ReplaceAllStringFunc(template, func(match string) string {
		name := match[2 : len(match)-2]
		if value, ok := resolve(name); ok {
			return value
		}
		return match
	})
}

func TemplateLayout(source string) string {
	at := sectionBreak.FindStringIndex(source)
	if at == nil {
		return source
	}
	return source[:at[0]]
}

var (
	moduleOpen  = regexp.MustCompile(`(?i)\[\[\s*module\s+([a-z0-9_]+)`)
	moduleClose = regexp.MustCompile(`(?i)^\[\[\s*/\s*module\s*]]`)
)

// A quoted value may hold ]], as in prependLine="[[#if ...]]". A quote left open
// falls back to the first ]], so a typo does not swallow the rest of the page.
func tagEnd(s string) int {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch {
		case inQuote && s[i] == '\\' && i+1 < len(s):
			i++
		case s[i] == '"':
			inQuote = !inQuote
		case !inQuote && strings.HasPrefix(s[i:], "]]"):
			return i + 2
		}
	}
	if i := strings.Index(s, "]]"); i >= 0 {
		return i + 2
	}
	return -1
}

type moduleTag struct {
	start, end int
	name       string
}

func nextModule(s string) (moduleTag, bool) {
	for from := 0; ; {
		open := moduleOpen.FindStringSubmatchIndex(s[from:])
		if open == nil {
			return moduleTag{}, false
		}
		start, nameEnd := from+open[0], from+open[1]
		if n := tagEnd(s[nameEnd:]); n >= 0 {
			return moduleTag{start: start, end: nameEnd + n, name: s[from+open[2] : from+open[3]]}, true
		}
		from = nameEnd
	}
}

// A body may hold modules with bodies of their own, so the end tag that closes
// it is the one that balances them, not the first one met.
func bodyEnd(s string, hasBody func(name string) bool) int {
	depth := 0
	for i := 0; i < len(s); {
		k := strings.Index(s[i:], "[[")
		if k < 0 {
			return -1
		}
		i += k
		if loc := moduleClose.FindStringIndex(s[i:]); loc != nil {
			if depth == 0 {
				return i + loc[1]
			}
			depth--
			i += loc[1]
			continue
		}
		if tag, ok := nextModule(s[i:]); ok && tag.start == 0 {
			if hasBody(tag.name) {
				depth++
			}
			i += tag.end
			continue
		}
		i += 2
	}
	return -1
}

func OutsideModules(source string, hasBody func(name string) bool, fill func(string) string) string {
	var out strings.Builder
	rest := source
	for {
		tag, ok := nextModule(rest)
		if !ok {
			out.WriteString(fill(rest))
			return out.String()
		}
		if !hasBody(tag.name) {
			out.WriteString(fill(rest[:tag.end]))
			rest = rest[tag.end:]
			continue
		}
		end := bodyEnd(rest[tag.end:], hasBody)
		if end < 0 {
			out.WriteString(fill(rest))
			return out.String()
		}
		out.WriteString(fill(rest[:tag.end]))
		out.WriteString(rest[tag.end : tag.end+end])
		rest = rest[tag.end+end:]
	}
}

// ThisVars is the pass an included page goes through. %%this|name%% reaches the
// page that pulled it in, and every other name is left standing for whatever
// pass claims it later.
func ThisVars(source string, vars *Vars) string {
	return ApplyTemplate(source, vars.This)
}

// PreRender is everything the source goes through on its way to the renderer.
// Both passes are blind to code blocks, the way a %% pair in prose already is.
func PreRender(source string, vars *Vars) string {
	return Galleries(Buttons(Inputs(ThisVars(source, vars))))
}

// PageVars is the pass the category template goes through, where a variable
// carries no prefix. index and total exist nowhere else.
func PageVars(template string, vars *Vars, index, total int) string {
	return ApplyTemplate(template, func(name string) (string, bool) {
		switch name {
		case "index":
			return strconv.Itoa(index), true
		case "total":
			return strconv.Itoa(total), true
		}
		return vars.Lookup(name)
	})
}
