// Package article turns a request path into the page it names and the
// parameters riding along with it.
package article

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const defaultHomePage = "main"

type Param = page.PathParam

type Params = page.PathParams

var (
	forumStart    = regexp.MustCompile(`^forum/start(.*)$`)
	forumCategory = regexp.MustCompile(`^forum/c-(\d+)(.*)$`)
	forumThread   = regexp.MustCompile(`^forum/t-(\d+)(.*)$`)
	forumSection  = regexp.MustCompile(`^forum/s-(\d+)(.*)$`)
)

// ParsePath reads the path with the leading slash already gone. homePage is
// the site's own setting, which only an empty name reaches for.
func ParsePath(raw, homePage string) (string, Params) {
	segments := strings.Split(rewriteForum(raw), "/")

	name := strings.TrimSpace(unquote(segments[0]))
	if name == "" {
		name = strings.TrimSpace(homePage)
	}
	if name == "" {
		name = defaultHomePage
	}

	var params Params
	rest := segments[1:]
	for i := 0; i < len(rest); i += 2 {
		key := strings.ToLower(unquote(rest[i]))
		value, bare := "", true
		if i+1 < len(rest) {
			value, bare = unquote(rest[i+1]), false
		}
		if key == "" && value == "" {
			continue
		}
		params = params.Put(Param{Key: key, Value: value, Bare: bare})
	}
	return name, params
}

// rewriteForum turns the four Wikidot forum URLs into the pages that answer
// them. Only the first match can apply, since each rewrite drops the prefix
// the others need.
func rewriteForum(path string) string {
	if m := forumStart.FindStringSubmatch(path); m != nil {
		return "forum:start" + m[1]
	}
	if m := forumCategory.FindStringSubmatch(path); m != nil {
		return "forum:category/c/" + m[1] + m[2]
	}
	if m := forumThread.FindStringSubmatch(path); m != nil {
		return "forum:thread/t/" + m[1] + m[2]
	}
	if m := forumSection.FindStringSubmatch(path); m != nil {
		return "forum:start/s/" + m[1] + m[2]
	}
	return path
}

// put keeps the position of the first occurrence, the way assigning to a
// Python dict does.
// Encode writes the parameters back as a path, sorted by key. Only the values
// are escaped, and only the first bare key survives.
func Encode(p Params) string {
	var named []Param
	bare := ""
	hasBare := false
	for _, param := range p {
		if param.Bare {
			if !hasBare {
				bare, hasBare = param.Key, true
			}
			continue
		}
		named = append(named, param)
	}
	slices.SortFunc(named, func(a, b Param) int { return strings.Compare(a.Key, b.Key) })

	var b strings.Builder
	for _, param := range named {
		b.WriteString("/" + param.Key + "/" + page.QuoteAll(param.Value))
	}
	if hasBare {
		b.WriteString("/" + bare)
	}
	return b.String()
}

// RedirectTarget is the Location a request gets when it names a page by
// anything other than the normalized name. ok is false when the name already
// is one.
func RedirectTarget(name string, params Params) (target string, ok bool) {
	normalized := wikidot.Normalize(name)
	if normalized == name {
		return "", false
	}
	// Escaped once more on the way out because Encode leaves the keys alone,
	// and a key may hold anything the path did.
	return iriToURI("/" + normalized + Encode(params)), true
}

// iriSafe is what Django leaves alone when it writes a Location, on top of the
// unreserved characters every escaper keeps. It spares the percent sign, so
// text escaped once does not grow a second time.
const iriSafe = "/#%[]=:;$&()+,!*?@'"

func iriToURI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b.WriteByte(c)
		case c == '_' || c == '.' || c == '-' || c == '~':
			b.WriteByte(c)
		case strings.IndexByte(iriSafe, c) >= 0:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
