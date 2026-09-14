import zhHans from '../locales/zh-hans.json'

type Catalog = Record<string, string>

const catalogs: Record<string, Catalog> = {
  'zh-hans': zhHans,
}

const FALLBACK = 'zh-hans'

let current = FALLBACK

export function setLanguage(lang: string) {
  current = lang in catalogs ? lang : FALLBACK
}

export function language(): string {
  return current
}

// A missing key falls back to the source language rather than to the key
// itself, because the bundle always carries zh-hans and showing Chinese beats
// showing search.total to a reader.
export function t(key: string, vars?: Record<string, string | number>): string {
  const text = catalogs[current]?.[key] ?? catalogs[FALLBACK][key] ?? key
  if (!vars) {
    return text
  }
  return text.replace(/\{(\w+)\}/g, (whole, name) => (name in vars ? String(vars[name]) : whole))
}
