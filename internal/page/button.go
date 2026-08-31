package page

import "regexp"

// A bare-word kind keeps a stray [[button in prose as prose, and arguments stop
// at the first ] because past it a block head cannot be told from the text.
var buttonPattern = regexp.MustCompile(`(?i)\[\[button\s+([A-Za-z][A-Za-z0-9_-]*)([^\]]*)\]\]`)

// This runs on the source rather than the tree, because the renderer knows
// modules and does not know buttons.
func Buttons(source string) string {
	return buttonPattern.ReplaceAllString(source, `[[module Button type="$1"$2]]`)
}
