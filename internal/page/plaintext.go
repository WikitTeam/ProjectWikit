package page

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const previewLength = 200

var (
	plainDropped = regexp.MustCompile(`(?is)\[\[(?:module\s+(?:css|listpages|countpages|listusers|frontforum|members|authorrank|nextpage|previouspage|time|pagedescription|gallerylist|input)|code|html|embed|iframe)\b[^\]]*\]\].*?\[\[/(?:module|code|html|embed|iframe)\]\]`)
	plainComment = regexp.MustCompile(`(?s)\[!--.*?--\]`)
	plainTriple  = regexp.MustCompile(`\[\[\[([^\]|]*)(?:\|([^\]]*))?\]\]\]`)
	plainLink    = regexp.MustCompile(`\[\*?(?:https?://|/|#)\S*\s+([^\]]+)\]`)
	plainColor   = regexp.MustCompile(`##[^|#\n]+\|([^#]*)##`)
	plainTag     = regexp.MustCompile(`\[\[[^\]]*\]\]`)
	plainLine    = regexp.MustCompile(`(?m)^(?:-{4,}[ \t]*$|={4,}[ \t]*$|\+{1,6}\s+|[*#]\s+|>+\s*|[ \t]+)`)
	plainMarks   = strings.NewReplacer("{{", "", "}}", "", "@@", "", "@<", "", ">@", "", "\\\n", " ")
	plainPairs   = []*regexp.Regexp{
		regexp.MustCompile(`()\*\*(.+?)\*\*`), regexp.MustCompile(`(^|[^:])//(.+?)//`),
		regexp.MustCompile(`()__(.+?)__`), regexp.MustCompile(`()--(.+?)--`),
		regexp.MustCompile(`(),,(.+?),,`), regexp.MustCompile(`()\^\^(.+?)\^\^`),
	}
	plainSpace  = regexp.MustCompile(`\s+`)
	headingLine = regexp.MustCompile(`^\s*\+{1,6}\s`)
	blankLines  = regexp.MustCompile(`\n\s*\n`)
)

// Text a reader would see, worked out from the source alone. Rendering every
// listed page just for a preview would cost far more than it is worth.
func plainText(source string) string {
	s := plainComment.ReplaceAllString(source, "")
	s = plainDropped.ReplaceAllString(s, "")
	s = plainTriple.ReplaceAllStringFunc(s, func(m string) string {
		parts := plainTriple.FindStringSubmatch(m)
		if strings.TrimSpace(parts[2]) != "" {
			return parts[2]
		}
		return parts[1]
	})
	s = plainLink.ReplaceAllString(s, "$1")
	s = plainColor.ReplaceAllString(s, "$1")
	s = plainTag.ReplaceAllString(s, "")
	s = plainLine.ReplaceAllString(s, "")
	for _, pair := range plainPairs {
		s = pair.ReplaceAllString(s, "${1}${2}")
	}
	return plainMarks.Replace(s)
}

func collapse(s string) string {
	return strings.TrimSpace(plainSpace.ReplaceAllString(s, " "))
}

func preview(source string, n int) string {
	text := collapse(plainText(source))
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:n])) + "..."
}

// A heading is not a paragraph, so the first block of running text wins.
func firstParagraph(source string) string {
	cleaned := plainDropped.ReplaceAllString(plainComment.ReplaceAllString(source, ""), "")
	for _, block := range blankLines.Split(cleaned, -1) {
		if headingLine.MatchString(block) {
			continue
		}
		if text := collapse(plainText(block)); text != "" {
			return text
		}
	}
	return ""
}

func previewLengthOf(name string) (int, bool) {
	arg, ok := strings.CutPrefix(name, "preview(")
	if !ok {
		return 0, false
	}
	arg, ok = strings.CutSuffix(arg, ")")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}
