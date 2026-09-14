package page

import (
	"regexp"
	"strings"
)

// A bare-word kind keeps a stray [[button in prose as prose, and arguments stop
// at the first ] because past it a block head cannot be told from the text.
var buttonPattern = regexp.MustCompile(`(?i)\[\[button\s+([A-Za-z][A-Za-z0-9_-]*)([^\]]*)\]\]`)

var argumentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*=`)

// This runs on the source rather than the tree, because the renderer knows
// modules and does not know buttons.
func Buttons(source string) string {
	return buttonPattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := buttonPattern.FindStringSubmatch(match)
		tags, rest := splitButtonArguments(parts[2])
		head := `[[module Button type="` + parts[1] + `"`
		if tags != "" {
			head += ` tags="` + tags + `"`
		}
		return head + rest + `]]`
	})
}

// The renderer throws away head words that are not key=value, so tag operations
// reach a module only once they are gathered into one argument.
func splitButtonArguments(tail string) (string, string) {
	var tags, rest strings.Builder
	for i := 0; i < len(tail); {
		if headSpace(tail[i]) {
			i++
			continue
		}
		start := i
		if loc := argumentName.FindStringIndex(tail[i:]); loc != nil {
			i += loc[1]
			if i < len(tail) && tail[i] == '"' {
				i++
				for i < len(tail) && tail[i] != '"' {
					i++
				}
				if i < len(tail) {
					i++
				}
			} else {
				for i < len(tail) && !headSpace(tail[i]) {
					i++
				}
			}
			rest.WriteByte(' ')
			rest.WriteString(tail[start:i])
			continue
		}
		for i < len(tail) && !headSpace(tail[i]) {
			i++
		}
		if tags.Len() > 0 {
			tags.WriteByte(' ')
		}
		// A quote here would end the gathered argument early and turn whatever
		// follows into arguments of its own.
		tags.WriteString(strings.ReplaceAll(tail[start:i], `"`, ""))
	}
	return tags.String(), rest.String()
}

func headSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
