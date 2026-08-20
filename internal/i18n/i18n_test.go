package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeCatalog(t *testing.T, dir, lang string, entries map[string]string) {
	t.Helper()
	raw, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("Marshal() err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.Join(dir, lang+fileSuffix), raw, 0o644); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
}

func load(t *testing.T, overrideDir string) *Bundle {
	t.Helper()
	b, err := Load(overrideDir)
	if err != nil {
		t.Fatalf("Load(%q) err = %v, want nil", overrideDir, err)
	}
	return b
}

func TestLoadReadsEmbeddedCatalog(t *testing.T) {
	l := load(t, "").Localizer(DefaultLanguage)
	if got := l.T("button-copy-clipboard"); got != "复制" {
		t.Errorf("T(%q) = %q, want %q", "button-copy-clipboard", got, "复制")
	}
}

func TestTReturnsIDWhenMissing(t *testing.T) {
	l := load(t, "").Localizer(DefaultLanguage)
	if got := l.T("no-such-id"); got != "no-such-id" {
		t.Errorf("T(%q) = %q, want %q", "no-such-id", got, "no-such-id")
	}
}

func TestTFallsBackToDefaultLanguage(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, "en", map[string]string{"toc-open": "Expand"})
	b := load(t, dir)

	l := b.Localizer("en")
	if got := l.T("toc-open"); got != "Expand" {
		t.Errorf("T(%q) = %q, want %q", "toc-open", got, "Expand")
	}
	if got := l.T("toc-close"); got != "关闭" {
		t.Errorf("T(%q) = %q, want fallback %q", "toc-close", got, "关闭")
	}
}

func TestTSubstitutesNamedArgs(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, DefaultLanguage, map[string]string{"greet": "{who} 有 {count} 条消息"})
	l := load(t, dir).Localizer(DefaultLanguage)

	if got := l.T("greet", "who", "Kakushi", "count", 3); got != "Kakushi 有 3 条消息" {
		t.Errorf("T(greet) = %q, want %q", got, "Kakushi 有 3 条消息")
	}
}

func TestTLeavesUnsuppliedPlaceholders(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, DefaultLanguage, map[string]string{"greet": "{who} 有 {count} 条消息"})
	l := load(t, dir).Localizer(DefaultLanguage)

	if got := l.T("greet", "who", "Kakushi"); got != "Kakushi 有 {count} 条消息" {
		t.Errorf("T(greet) = %q, want %q", got, "Kakushi 有 {count} 条消息")
	}
}

func TestTIgnoresTrailingOddArg(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, DefaultLanguage, map[string]string{"greet": "{who} 来了"})
	l := load(t, dir).Localizer(DefaultLanguage)

	if got := l.T("greet", "who", "Kakushi", "count"); got != "Kakushi 来了" {
		t.Errorf("T(greet) = %q, want %q", got, "Kakushi 来了")
	}
}

func TestLoadMergesOverrideDir(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, DefaultLanguage, map[string]string{"toc-open": "打开"})
	l := load(t, dir).Localizer(DefaultLanguage)

	if got := l.T("toc-open"); got != "打开" {
		t.Errorf("T(%q) = %q, want override %q", "toc-open", got, "打开")
	}
	if got := l.T("toc-close"); got != "关闭" {
		t.Errorf("T(%q) = %q, want builtin %q", "toc-close", got, "关闭")
	}
}

func TestLoadIgnoresMissingOverrideDir(t *testing.T) {
	l := load(t, filepath.Join(t.TempDir(), "missing")).Localizer(DefaultLanguage)
	if got := l.T("toc-close"); got != "关闭" {
		t.Errorf("T(%q) = %q, want %q", "toc-close", got, "关闭")
	}
}

func TestLoadRejectsBadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en"+fileSuffix), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load() err = nil, want non-nil")
	}
}

func TestLocalizerFallsBackForUnknownLanguage(t *testing.T) {
	l := load(t, "").Localizer("de")
	if l.Lang() != DefaultLanguage {
		t.Errorf("Lang() = %q, want %q", l.Lang(), DefaultLanguage)
	}
}

func TestLocalizerNormalizesLanguageTag(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, "en", map[string]string{"toc-open": "Expand"})
	b := load(t, dir)

	if got := b.Localizer(" EN ").Lang(); got != "en" {
		t.Errorf("Localizer(%q).Lang() = %q, want %q", " EN ", got, "en")
	}
}

func TestBundleLanguages(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, "en", map[string]string{"toc-open": "Expand"})
	got := load(t, dir).Languages()

	if !slices.Equal(got, []string{"en", "zh-hans"}) {
		t.Errorf("Languages() = %v, want %v", got, []string{"en", "zh-hans"})
	}
}
