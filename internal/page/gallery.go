package page

import (
	"regexp"
	"strings"
)

// The name has to end at the head, or [[galleryish]] would be rewritten into a
// module that does not exist.
var (
	galleryPattern = regexp.MustCompile(`(?i)\[\[gallery(\s[^\]]*)?\]\]`)
	galleryClose   = regexp.MustCompile(`(?i)\n[ \t]*\[\[/gallery\]\]`)
)

func Galleries(source string) string {
	var out strings.Builder
	rest := source
	for {
		closing := galleryClose.FindStringIndex(rest)
		if closing == nil {
			break
		}
		heads := galleryPattern.FindAllStringSubmatchIndex(rest[:closing[0]], -1)
		if len(heads) == 0 {
			out.WriteString(rest[:closing[1]])
			rest = rest[closing[1]:]
			continue
		}
		head := heads[len(heads)-1]
		body, ok := strings.CutPrefix(strings.TrimLeft(rest[head[1]:closing[0]], " \t"), "\n")
		if !ok {
			out.WriteString(galleryPattern.ReplaceAllString(rest[:closing[1]], `[[module Gallery$1]]`))
			rest = rest[closing[1]:]
			continue
		}
		params := ""
		if head[2] >= 0 {
			params = rest[head[2]:head[3]]
		}
		out.WriteString(galleryPattern.ReplaceAllString(rest[:head[0]], `[[module Gallery$1]]`))
		out.WriteString("[[module GalleryList" + params + "]]\n" + body + "\n[[/module]]")
		rest = rest[closing[1]:]
	}
	out.WriteString(galleryPattern.ReplaceAllString(rest, `[[module Gallery$1]]`))
	return out.String()
}
