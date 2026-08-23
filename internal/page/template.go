// Package page holds the state of one page render and the substitutions that
// run before ftml sees the source.
package page

import "regexp"

// Go's dot excludes newlines by default, the same as Python's without DOTALL,
// so a %% pair may not span lines.
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
