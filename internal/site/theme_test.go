package site

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestThemeURLWithoutTheme(t *testing.T) {
	if got := ThemeURL(nil); got != "" {
		t.Errorf("ThemeURL(nil) = %q, want %q", got, "")
	}
}

func TestThemeURLOfInlineTheme(t *testing.T) {
	theme := &db.Theme{Slug: "dark", Mode: db.ThemeInline, UpdatedAt: time.Unix(1700000000, 0)}
	want := "/-/theme/dark.css?v=1700000000"
	if got := ThemeURL(theme); got != want {
		t.Errorf("ThemeURL(inline) = %q, want %q", got, want)
	}
}

func TestThemeURLOfExternalTheme(t *testing.T) {
	theme := &db.Theme{Slug: "dark", Mode: db.ThemeExternal, ExternalURL: "  https://cdn.example/x.css  "}
	want := "https://cdn.example/x.css"
	if got := ThemeURL(theme); got != want {
		t.Errorf("ThemeURL(external) = %q, want %q", got, want)
	}
}

func TestThemeURLOfExternalThemeWithoutURL(t *testing.T) {
	theme := &db.Theme{Slug: "dark", Mode: db.ThemeExternal, ExternalURL: "   "}
	if got := ThemeURL(theme); got != "" {
		t.Errorf("ThemeURL(external without url) = %q, want %q", got, "")
	}
}
