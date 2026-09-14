package page

import "regexp"

var inputPattern = regexp.MustCompile(`(?is)\[\[input(\s[^\]]*)?]](.*?)\[\[/input]]`)

func Inputs(source string) string {
	return inputPattern.ReplaceAllString(source, "[[module Input$1]]$2[[/module]]")
}
