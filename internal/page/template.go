// Package page holds the state of one page render and the substitutions that
// run before ftml sees the source.
package page

import (
	"regexp"
	"strconv"
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

// ThisVars is the pass an included page goes through. %%this|name%% reaches the
// page that pulled it in, and every other name is left standing for whatever
// pass claims it later.
func ThisVars(source string, vars *Vars) string {
	return ApplyTemplate(source, vars.This)
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
