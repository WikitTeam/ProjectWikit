package difftest

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

type Kind string

const (
	KindStatus Kind = "status"
	KindHeader Kind = "header"
	KindBody   Kind = "body"
)

type Diff struct {
	Kind Kind
	Name string
	A    string
	B    string
}

func (d Diff) String() string {
	if d.Name == "" {
		return fmt.Sprintf("%s: a=%s b=%s", d.Kind, d.A, d.B)
	}
	return fmt.Sprintf("%s %s: a=%s b=%s", d.Kind, d.Name, d.A, d.B)
}

type Scrubber struct {
	Name    string
	Pattern *regexp.Regexp
	Repl    string
}

var DefaultScrubbers = []Scrubber{
	{
		Name:    "csrfmiddlewaretoken",
		Pattern: regexp.MustCompile(`(name="csrfmiddlewaretoken"\s+value=")[^"]*(")`),
		Repl:    "${1}SCRUBBED${2}",
	},
	{
		Name:    "html-lang",
		Pattern: regexp.MustCompile(`(<html lang="|<meta http-equiv="content-language" content=")zh(-hans)?(")`),
		Repl:    "${1}SCRUBBED${3}",
	},
}

var DefaultIgnoredHeaders = []string{
	"Date",
	"Server",
	"Content-Length",
	"Connection",
	"Keep-Alive",
	"Transfer-Encoding",
}

type Comparer struct {
	IgnoreHeaders []string
	Scrubbers     []Scrubber
}

func NewComparer() *Comparer {
	return &Comparer{
		IgnoreHeaders: slices.Clone(DefaultIgnoredHeaders),
		Scrubbers:     slices.Clone(DefaultScrubbers),
	}
}

type Result struct {
	Diffs  []Diff
	Scrubs map[string]int
}

func (r Result) Same() bool { return len(r.Diffs) == 0 }

func (r Result) String() string {
	if r.Same() {
		return "same"
	}
	lines := make([]string, 0, len(r.Diffs))
	for _, d := range r.Diffs {
		lines = append(lines, d.String())
	}
	return strings.Join(lines, "\n")
}

func (c *Comparer) Compare(a, b Response) Result {
	res := Result{Scrubs: make(map[string]int)}

	if a.Status != b.Status {
		res.Diffs = append(res.Diffs, Diff{
			Kind: KindStatus,
			A:    fmt.Sprint(a.Status),
			B:    fmt.Sprint(b.Status),
		})
	}

	res.Diffs = append(res.Diffs, c.compareHeaders(a.Header, b.Header)...)

	scrubbedA := c.scrub(a.Body, res.Scrubs)
	scrubbedB := c.scrub(b.Body, res.Scrubs)
	if !bytes.Equal(scrubbedA, scrubbedB) {
		res.Diffs = append(res.Diffs, bodyDiff(scrubbedA, scrubbedB))
	}
	return res
}

func (c *Comparer) scrub(body []byte, hits map[string]int) []byte {
	out := body
	for _, s := range c.Scrubbers {
		n := len(s.Pattern.FindAll(out, -1))
		if n == 0 {
			continue
		}
		hits[s.Name] += n
		out = s.Pattern.ReplaceAll(out, []byte(s.Repl))
	}
	return out
}

func (c *Comparer) compareHeaders(a, b http.Header) []Diff {
	ignored := make(map[string]bool, len(c.IgnoreHeaders))
	for _, name := range c.IgnoreHeaders {
		ignored[http.CanonicalHeaderKey(name)] = true
	}

	names := make([]string, 0, len(a)+len(b))
	for name := range a {
		names = append(names, name)
	}
	for name := range b {
		if _, ok := a[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	var diffs []Diff
	for _, name := range names {
		if ignored[name] {
			continue
		}
		valueA, valueB := headerValue(a, name), headerValue(b, name)
		if valueA != valueB {
			diffs = append(diffs, Diff{Kind: KindHeader, Name: name, A: valueA, B: valueB})
		}
	}
	return diffs
}

func headerValue(h http.Header, name string) string {
	values := h.Values(name)
	if name == "Set-Cookie" {
		names := make([]string, 0, len(values))
		for _, v := range values {
			cookieName, _, _ := strings.Cut(v, "=")
			names = append(names, strings.TrimSpace(cookieName))
		}
		slices.Sort(names)
		return strings.Join(names, ", ")
	}
	return strings.Join(values, ", ")
}

func bodyDiff(a, b []byte) Diff {
	linesA := strings.Split(string(a), "\n")
	linesB := strings.Split(string(b), "\n")

	i := 0
	for i < len(linesA) && i < len(linesB) && linesA[i] == linesB[i] {
		i++
	}
	return Diff{
		Kind: KindBody,
		Name: fmt.Sprintf("line %d of %d/%d", i+1, len(linesA), len(linesB)),
		A:    excerpt(linesA, i),
		B:    excerpt(linesB, i),
	}
}

const excerptLimit = 120

func excerpt(lines []string, i int) string {
	if i >= len(lines) {
		return "<eof>"
	}
	line := lines[i]
	if len(line) > excerptLimit {
		return line[:excerptLimit] + "…"
	}
	return line
}
