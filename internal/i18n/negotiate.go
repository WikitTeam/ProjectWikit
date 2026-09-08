package i18n

import (
	"context"
	"fmt"

	"golang.org/x/text/language"
)

const AcceptHeader = "Accept-Language"

type languageKey struct{}

func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, languageKey{}, Normalize(lang))
}

// LanguageFrom is empty for a request that never met the negotiator, which
// Localizer already reads as the default language.
func LanguageFrom(ctx context.Context) string {
	lang, _ := ctx.Value(languageKey{}).(string)
	return lang
}

func (b *Bundle) For(ctx context.Context) *Localizer {
	return b.Localizer(LanguageFrom(ctx))
}

// A chosen language outranks the browser's, because a browser speaks for a
// device and a member speaks for themselves.
func (b *Bundle) Negotiate(chosen, accept, siteDefault string) string {
	if b.Has(chosen) {
		return Normalize(chosen)
	}
	if lang := b.Match(accept); lang != "" {
		return lang
	}
	if b.Has(siteDefault) {
		return Normalize(siteDefault)
	}
	return DefaultLanguage
}

// Match is empty when the header names nothing the bundle carries, so the
// caller can tell "no opinion" apart from "asked for the default".
func (b *Bundle) Match(accept string) string {
	if accept == "" {
		return ""
	}
	wanted, _, err := language.ParseAcceptLanguage(accept)
	if err != nil || len(wanted) == 0 {
		return ""
	}
	_, index, conf := b.matcher.Match(wanted...)
	if conf == language.No || index >= len(b.tagged) {
		return ""
	}
	return b.tagged[index]
}

type Choice struct {
	Tag  string
	Name string
}

func (b *Bundle) Choices() []Choice {
	out := make([]Choice, 0, len(b.tagged))
	for _, tag := range b.tagged {
		out = append(out, Choice{Tag: tag, Name: b.Name(tag)})
	}
	return out
}

// A catalog names its own language, so adding one is a JSON file and nothing
// else.
func (b *Bundle) Name(lang string) string {
	lang = Normalize(lang)
	if name := b.catalogs[lang][nameKey]; name != "" {
		return name
	}
	return lang
}

const nameKey = "language-name"

// The default language goes first because that is the tag a matcher answers
// with when it recognises nothing.
func (b *Bundle) buildMatcher() error {
	names := b.Languages()
	b.tagged = append([]string{DefaultLanguage}, remove(names, DefaultLanguage)...)
	tags := make([]language.Tag, 0, len(b.tagged))
	for _, name := range b.tagged {
		tag, err := language.Parse(name)
		if err != nil {
			return fmt.Errorf("catalog %q is not a language tag: %w", name, err)
		}
		tags = append(tags, tag)
	}
	b.matcher = language.NewMatcher(tags)
	return nil
}

func remove(names []string, drop string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if name != drop {
			out = append(out, name)
		}
	}
	return out
}
