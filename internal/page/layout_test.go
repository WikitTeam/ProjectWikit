package page

import (
	"strings"
	"testing"
)

func TestTemplateLayout(t *testing.T) {
	cases := map[string]string{
		"top\n====\n[[form]]\n[[/form]]": "top\n",
		"top\n=====\nbottom":             "top\n",
		"top\n==== \r\nbottom":           "top\n",
		"no break here":                  "no break here",
		"a ==== b\nc":                    "a ==== b\nc",
	}
	for in, want := range cases {
		if got := TemplateLayout(in); got != want {
			t.Errorf("TemplateLayout(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOutsideModules(t *testing.T) {
	hasBody := func(name string) bool { return name == "ListPages" || name == "listpages" }
	upper := func(s string) string { return strings.ReplaceAll(s, "x", "X") }
	cases := map[string]string{
		"x [[module ListPages a=\"x\"]]x[[/module]] x":         "X [[module ListPages a=\"X\"]]x[[/module]] X",
		"x [[module Rate]] x [[module listpages]]x[[/module]]": "X [[module Rate]] X [[module listpages]]x[[/module]]",
		"x [[module listpages]] x":                             "X [[module listpages]] X",
		"x":                                                    "X",
	}
	for in, want := range cases {
		if got := OutsideModules(in, hasBody, upper); got != want {
			t.Errorf("OutsideModules(%q) = %q, want %q", in, got, want)
		}
	}
}
