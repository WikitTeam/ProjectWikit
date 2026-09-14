package page

import "regexp"

// The name has to end at the head, or [[galleryish]] would be rewritten into a
// module that does not exist.
var galleryPattern = regexp.MustCompile(`(?i)\[\[gallery(\s[^\]]*)?\]\]`)

func Galleries(source string) string {
	return galleryPattern.ReplaceAllString(source, `[[module Gallery$1]]`)
}
