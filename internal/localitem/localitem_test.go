package localitem

import (
	"strings"
	"testing"
)

func TestSplitTakesTwoSegments(t *testing.T) {
	cases := []struct {
		prefix string
		path   string
		name   string
		rest   string
	}{
		{CodePrefix, "/local--code/probe:full/1", "probe:full", "1"},
		{ThemePrefix, "/local--theme/probe:full/style.css", "probe:full", "style.css"},
		{HTMLPrefix, "/local--html/probe:full/abc-def", "probe:full", "abc-def"},
	}
	for _, c := range cases {
		name, rest, ok := split(c.path, c.prefix)
		if !ok || name != c.name || rest != c.rest {
			t.Errorf("split(%q) = %q, %q, %v, want %q, %q, true", c.path, name, rest, ok, c.name, c.rest)
		}
	}
}

func TestSplitRejectsOtherShapes(t *testing.T) {
	for _, path := range []string{
		"/local--code/",
		"/local--code/probe:full",
		"/local--code/probe:full/",
		"/local--code//1",
		"/local--code/probe:full/1/2",
		"/local--files/probe:full/1",
	} {
		if _, _, ok := split(path, "/local--code/"); ok {
			t.Errorf("split(%q) ok = true, want false", path)
		}
	}
}

func TestCodeMime(t *testing.T) {
	cases := map[string]string{
		"html":       "text/html; charset=utf-8",
		"XHTML":      "text/html; charset=utf-8",
		"js":         "text/javascript; charset=utf-8",
		"javascript": "text/javascript; charset=utf-8",
		"jsx":        "text/javascript; charset=utf-8",
		"xml":        "application/xml; charset=utf-8",
		"css":        "text/css; charset=utf-8",
		"plain":      "text/plain; charset=utf-8",
		"":           "text/plain; charset=utf-8",
	}
	for language, want := range cases {
		if got := codeMime(language); got != want {
			t.Errorf("codeMime(%q) = %q, want %q", language, got, want)
		}
	}
}

func TestStripNoInclude(t *testing.T) {
	cases := map[string]string{
		"a[[noinclude]]b[[/noinclude]]c":                             "ac",
		"a[[noinclude]]b[[/noinclude]]c[[noinclude]]d[[/noinclude]]": "ac",
		"plain":                          "plain",
		"a[[noinclude]]b":                "a",
		"[[noinclude]]b[[/noinclude]]":   "",
		"a[[/noinclude]]b[[noinclude]]c": "a[[/noinclude]]b",
		"[[noinclude]]b[[/noinclude]]c":  "c",
	}
	for source, want := range cases {
		if got := stripNoInclude(source); got != want {
			t.Errorf("stripNoInclude(%q) = %q, want %q", source, got, want)
		}
	}
}

func TestStripNoIncludeManyPairs(t *testing.T) {
	source := "x" + strings.Repeat("[[noinclude]]a[[/noinclude]]", 1<<16) + "y"
	if got := stripNoInclude(source); got != "xy" {
		t.Errorf("stripNoInclude(%d pairs) = %q, want %q", 1<<16, got, "xy")
	}
}

func TestExpandParams(t *testing.T) {
	cases := []struct {
		source string
		params map[string]string
		want   string
	}{
		{"color: {$c};", map[string]string{"c": "red"}, "color: red;"},
		{"{$a} {$b}", map[string]string{"a": "{$b}", "b": "x"}, "{$b} x"},
		{"{$b} {$a}", map[string]string{"a": "{$b}", "b": "x"}, "x {$b}"},
		{"{$missing}", map[string]string{"c": "red"}, "{$missing}"},
		{"plain", nil, "plain"},
	}
	for _, c := range cases {
		got, ok := expandParams(c.source, c.params)
		if !ok || got != c.want {
			t.Errorf("expandParams(%q, %v) = %q, %v, want %q, true", c.source, c.params, got, ok, c.want)
		}
	}
}

func TestExpandParamsRefusesLargeGrowth(t *testing.T) {
	source := strings.Repeat("{$a}", 1024)
	value := strings.Repeat("x", 4096)
	if got, ok := expandParams(source, map[string]string{"a": value}); ok {
		t.Errorf("expandParams(1024 tokens, 4096 bytes) = %d bytes, true, want false", len(got))
	}
	if _, ok := expandParams("{$a}", map[string]string{"a": value}); !ok {
		t.Errorf("expandParams(1 token, 4096 bytes) ok = false, want true")
	}
}

func TestBlockByHash(t *testing.T) {
	blocks := []string{"<b>one</b>", "<b>two</b>"}
	const secondHash = "e64f30f9218c6b2848f37c7dfb36593b"

	got, ok := blockByHash(blocks, secondHash)
	if !ok || got != "<b>two</b>" {
		t.Errorf("blockByHash(%q) = %q, %v, want %q, true", secondHash, got, ok, "<b>two</b>")
	}
	if got, ok := blockByHash(blocks, "00000000000000000000000000000000"); ok {
		t.Errorf("blockByHash(unknown) = %q, true, want \"\", false", got)
	}
}

func TestStringMap(t *testing.T) {
	got := stringMap(`{"a": "b"}`)
	if len(got) != 1 || got["a"] != "b" {
		t.Errorf("stringMap() = %v, want map[a:b]", got)
	}
	for _, raw := range []string{"", "[]", `{"a": 1}`} {
		if got := stringMap(raw); got != nil {
			t.Errorf("stringMap(%q) = %v, want nil", raw, got)
		}
	}
}
