package i18n

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
)

//go:embed locales/*.json
var embedded embed.FS

const (
	DefaultLanguage = "zh-hans"
	embeddedDir     = "locales"
	fileSuffix      = ".json"
)

type Bundle struct {
	catalogs map[string]map[string]string
}

func Load(overrideDir string) (*Bundle, error) {
	b := &Bundle{catalogs: make(map[string]map[string]string)}
	if err := b.merge(embedded, embeddedDir); err != nil {
		return nil, err
	}
	if overrideDir != "" {
		if err := b.merge(os.DirFS(overrideDir), "."); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if len(b.catalogs[DefaultLanguage]) == 0 {
		return nil, fmt.Errorf("catalog for default language %q is empty", DefaultLanguage)
	}
	return b, nil
}

func (b *Bundle) merge(fsys fs.FS, dir string) error {
	names, err := fs.Glob(fsys, path.Join(dir, "*"+fileSuffix))
	if err != nil {
		return err
	}
	for _, name := range names {
		lang := Normalize(strings.TrimSuffix(path.Base(name), fileSuffix))
		if lang == "" {
			return fmt.Errorf("catalog %s: filename is not a language tag", name)
		}
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		var entries map[string]string
		if err := json.Unmarshal(raw, &entries); err != nil {
			return fmt.Errorf("parse catalog %s: %w", name, err)
		}
		catalog, ok := b.catalogs[lang]
		if !ok {
			catalog = make(map[string]string, len(entries))
			b.catalogs[lang] = catalog
		}
		for id, text := range entries {
			catalog[id] = text
		}
	}
	return nil
}

func (b *Bundle) Languages() []string {
	langs := make([]string, 0, len(b.catalogs))
	for lang := range b.catalogs {
		langs = append(langs, lang)
	}
	slices.Sort(langs)
	return langs
}

func (b *Bundle) Has(lang string) bool {
	_, ok := b.catalogs[Normalize(lang)]
	return ok
}

func (b *Bundle) Localizer(lang string) *Localizer {
	lang = Normalize(lang)
	if !b.Has(lang) {
		lang = DefaultLanguage
	}
	return &Localizer{bundle: b, lang: lang}
}

type Localizer struct {
	bundle *Bundle
	lang   string
}

func (l *Localizer) Lang() string { return l.lang }

func (l *Localizer) T(id string, args ...any) string {
	text, ok := l.bundle.catalogs[l.lang][id]
	if !ok {
		text, ok = l.bundle.catalogs[DefaultLanguage][id]
	}
	if !ok {
		return id
	}
	return expand(text, args)
}

func Normalize(lang string) string {
	return strings.ToLower(strings.TrimSpace(lang))
}

func expand(text string, args []any) string {
	if len(args) < 2 || !strings.Contains(text, "{") {
		return text
	}
	pairs := make([]string, 0, len(args))
	for i := 0; i+1 < len(args); i += 2 {
		name, ok := args[i].(string)
		if !ok {
			continue
		}
		pairs = append(pairs, "{"+name+"}", fmt.Sprint(args[i+1]))
	}
	if len(pairs) == 0 {
		return text
	}
	return strings.NewReplacer(pairs...).Replace(text)
}
