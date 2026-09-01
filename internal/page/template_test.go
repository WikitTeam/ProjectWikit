package page

import "testing"

func testResolver(name string) (string, bool) {
	value, ok := map[string]string{
		"a":          "A",
		"this|title": "T",
		"":           "EMPTY",
	}[name]
	return value, ok
}

func TestApplyTemplate(t *testing.T) {
	tests := []struct{ in, want string }{
		{"%%a%%", "A"},
		{"x%%a%%y", "xAy"},
		{"%%a%%b%%c%%", "Ab%%c%%"},
		{"%%%%", "EMPTY"},
		{"%%%%%%", "EMPTY%%"},
		{"%%a", "%%a"},
		{"a%%", "a%%"},
		{"%%unknown%%", "%%unknown%%"},
		{"%%A%%", "%%A%%"},
		{"%%a\nb%%", "%%a\nb%%"},
		{"%%this|title%%", "T"},
		{"%%a%%%%a%%", "AA"},
		{"%%%a%%", "%%%a%%"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := ApplyTemplate(tt.in, testResolver); got != tt.want {
				t.Errorf("ApplyTemplate(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestApplyTemplateResolvesEachOccurrence(t *testing.T) {
	calls := 0
	got := ApplyTemplate("%%a%% %%a%%", func(string) (string, bool) {
		calls++
		return "x", true
	})

	if got != "x x" {
		t.Errorf("ApplyTemplate(%q) = %q, want %q", "%%a%% %%a%%", got, "x x")
	}
	if calls != 2 {
		t.Errorf("resolver calls = %d, want 2", calls)
	}
}

func TestApplyTemplateEmptyValue(t *testing.T) {
	got := ApplyTemplate("[%%a%%]", func(string) (string, bool) { return "", true })

	if got != "[]" {
		t.Errorf("ApplyTemplate(%q) = %q, want %q", "[%%a%%]", got, "[]")
	}
}

func TestInputsRewrite(t *testing.T) {
	got := Inputs("[[input]]\n# a\n * type: checkbox\n[[/input]]")
	want := "[[module Input]]\n# a\n * type: checkbox\n[[/module]]"
	if got != want {
		t.Errorf("Inputs() = %q, want %q", got, want)
	}
}

func TestInputsKeepsTheHead(t *testing.T) {
	got := Inputs(`[[input class="x"]]body[[/input]]`)
	if want := `[[module Input class="x"]]body[[/module]]`; got != want {
		t.Errorf("Inputs() = %q, want %q", got, want)
	}
}

func TestInputsLeavesOtherText(t *testing.T) {
	for _, in := range []string{"[[inputs]]x[[/inputs]]", "[[input]]no closing tag", "plain"} {
		if got := Inputs(in); got != in {
			t.Errorf("Inputs(%q) = %q, want it unchanged", in, got)
		}
	}
}
