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
