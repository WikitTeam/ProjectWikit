package i18n

import (
	"context"
	"testing"
)

func english(t *testing.T) *Bundle {
	t.Helper()
	dir := t.TempDir()
	writeCatalog(t, dir, "en", map[string]string{"toc-open": "Expand"})
	return load(t, dir)
}

func TestMatchReadsAcceptLanguage(t *testing.T) {
	b := english(t)
	for _, c := range []struct{ accept, want string }{
		{"en", "en"},
		{"en-US,en;q=0.9", "en"},
		{"fr;q=0.8,en;q=0.9", "en"},
		{"zh-CN,zh;q=0.9", "zh-hans"},
	} {
		if got := b.Match(c.accept); got != c.want {
			t.Errorf("Match(%q) = %q, want %q", c.accept, got, c.want)
		}
	}
}

func TestMatchIsEmptyWithoutAKnownLanguage(t *testing.T) {
	b := english(t)
	for _, accept := range []string{"", "de", "garbage!!"} {
		if got := b.Match(accept); got != "" {
			t.Errorf("Match(%q) = %q, want %q", accept, got, "")
		}
	}
}

func TestNegotiatePrefersTheChosenLanguage(t *testing.T) {
	if got := english(t).Negotiate("en", "zh-CN", DefaultLanguage); got != "en" {
		t.Errorf("Negotiate(%q, %q, %q) = %q, want %q", "en", "zh-CN", DefaultLanguage, got, "en")
	}
}

func TestNegotiateAsksTheBrowserWhenNothingWasChosen(t *testing.T) {
	if got := english(t).Negotiate("", "en-GB", DefaultLanguage); got != "en" {
		t.Errorf("Negotiate(%q, %q, %q) = %q, want %q", "", "en-GB", DefaultLanguage, got, "en")
	}
}

func TestNegotiateFallsBackToTheSiteLanguage(t *testing.T) {
	if got := english(t).Negotiate("", "de", "en"); got != "en" {
		t.Errorf("Negotiate(%q, %q, %q) = %q, want %q", "", "de", "en", got, "en")
	}
}

func TestNegotiateIgnoresLanguagesTheBundleLacks(t *testing.T) {
	if got := english(t).Negotiate("de", "de", "de"); got != DefaultLanguage {
		t.Errorf("Negotiate(%q, %q, %q) = %q, want %q", "de", "de", "de", got, DefaultLanguage)
	}
}

func TestForReadsTheLanguageOffTheContext(t *testing.T) {
	b := english(t)
	ctx := WithLanguage(context.Background(), "EN")
	if got := b.For(ctx).Lang(); got != "en" {
		t.Errorf("For(ctx).Lang() = %q, want %q", got, "en")
	}
}

func TestForFallsBackWithoutALanguageOnTheContext(t *testing.T) {
	if got := english(t).For(context.Background()).Lang(); got != DefaultLanguage {
		t.Errorf("For(ctx).Lang() = %q, want %q", got, DefaultLanguage)
	}
}

func TestLoadRejectsACatalogThatIsNotALanguageTag(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, "not+a+tag", map[string]string{"toc-open": "Expand"})
	if _, err := Load(dir); err == nil {
		t.Error("Load() err = nil, want non-nil")
	}
}
