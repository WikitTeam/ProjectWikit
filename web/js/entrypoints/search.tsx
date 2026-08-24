import { t } from '~util/i18n'
import * as React from 'react'
import Trans from '~util/trans'
import { useEffect, useRef, useState } from 'react'
import { searchModule, SearchResultItem } from '../api/search-module'
import { highlightWords } from '../reactive/pages/search/Search.utils'
import useConstCallback from '../util/const-callback'

interface Props {
  placeholder?: string
  tags?: string
  category?: string
  q?: string
  author?: string
  datefrom?: string
  dateto?: string
}

function parseTime(s: string | null): Date | null {
  if (!s) return null
  let str = s.trim().replace(' ', 'T')
  if (!/[zZ]|[+-]\d{2}:?\d{2}$/.test(str)) str += 'Z'
  const d = new Date(str)
  return isNaN(d.getTime()) ? null : d
}

function relTime(s: string | null): string {
  const d = parseTime(s)
  if (!d) return ''
  let diff = Math.floor((Date.now() - d.getTime()) / 1000)
  if (diff < 0) diff = 0
  const year = 31536000,
    month = 2592000,
    day = 86400,
    hour = 3600,
    min = 60
  if (diff >= year) return t('search.years-ago', { count: Math.floor(diff / year) })
  if (diff >= month) return t('search.months-ago', { count: Math.floor(diff / month) })
  if (diff >= day) return t('search.days-ago', { count: Math.floor(diff / day) })
  if (diff >= hour) return t('search.hours-ago', { count: Math.floor(diff / hour) })
  if (diff >= min) return t('search.minutes-ago', { count: Math.floor(diff / min) })
  return t('search.just-now')
}

function preciseTime(s: string | null): string {
  const d = parseTime(s)
  return d ? d.toLocaleString() : ''
}

const SearchModule: React.FC<Props> = ({
  placeholder,
  tags: defaultTags,
  q: initialQ,
  author: initialAuthor,
  datefrom: initialDateFrom,
  dateto: initialDateTo,
}) => {
  const [q, setQ] = useState(initialQ || '')
  const [author, setAuthor] = useState(initialAuthor || '')
  const [tags, setTags] = useState(defaultTags || '')
  const [dateFrom, setDateFrom] = useState(initialDateFrom || '')
  const [dateTo, setDateTo] = useState(initialDateTo || '')

  const [results, setResults] = useState<SearchResultItem[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [searched, setSearched] = useState(false)

  const seq = useRef(0)

  const runSearch = useConstCallback(async (offset: number, append: boolean) => {
    const my = ++seq.current
    if (append) setLoadingMore(true)
    else setLoading(true)
    setError(null)
    try {
      const resp = await searchModule({
        q: q.trim(),
        author: author.trim(),
        tags: tags.trim(),
        datefrom: dateFrom,
        dateto: dateTo,
        offset,
      })
      if (my !== seq.current) return
      setResults(prev => (append ? [...prev, ...resp.results] : resp.results))
      setHasMore(resp.hasMore)
      setTotal(resp.total)
      setSearched(true)
    } catch (e: any) {
      if (my !== seq.current) return
      setError(e?.message || t('search.failed'))
    } finally {
      if (my === seq.current) {
        setLoading(false)
        setLoadingMore(false)
      }
    }
  })

  useEffect(() => {
    const hasCriteria = !!(q.trim() || author.trim() || tags.trim() || dateFrom || dateTo)
    if (!hasCriteria) {
      seq.current++
      setResults([])
      setHasMore(false)
      setTotal(0)
      setSearched(false)
      setError(null)
      return
    }
    const t = setTimeout(() => runSearch(0, false), 300)
    return () => clearTimeout(t)
  }, [q, author, tags, dateFrom, dateTo])

  const onLoadMore = useConstCallback(() => runSearch(results.length, true))

  return (
    <div className="w-search">
      <div className="w-search-box">
        <input
          className="w-search-input"
          type="text"
          value={q}
          placeholder={placeholder || t('search.placeholder')}
          onChange={e => setQ(e.target.value)}
        />
      </div>

      <div className="w-search-filters">
        <input className="w-search-filter" type="text" value={author} placeholder={t('search.author-placeholder')} onChange={e => setAuthor(e.target.value)} />
        <input className="w-search-filter" type="text" value={tags} placeholder={t('search.tags-placeholder')} onChange={e => setTags(e.target.value)} />
        <div className="w-search-dates">
          <input className="w-search-filter w-search-date" type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)} />
          <span className="w-search-date-sep">–</span>
          <input className="w-search-filter w-search-date" type="date" value={dateTo} onChange={e => setDateTo(e.target.value)} />
        </div>
      </div>

      {error && <div className="w-search-error">{error}</div>}

      {loading ? (
        <div className="w-search-results">
          {[0, 1, 2, 3, 4].map(i => (
            <div className="w-search-skeleton" key={i}>
              <div className="w-search-sk-line w-search-sk-title" />
              <div className="w-search-sk-line" />
              <div className="w-search-sk-line w-search-sk-short" />
            </div>
          ))}
        </div>
      ) : searched && !results.length && !error ? (
        <div className="w-search-empty">{t('search.empty')}</div>
      ) : (
        <>
          {!!results.length && <div className="w-search-total">{t('search.total', { total })}</div>}
          <div className="w-search-results">
            {results.map((r, i) => (
              <div className="w-search-result" key={`${r.url}-${i}`}>
                <a className="w-search-result-title" href={r.url}>
                  {highlightWords(r.title, r.words)}
                </a>
                <div className="w-search-result-excerpt">{highlightWords(r.excerpt, r.words)}</div>
                <div className="w-search-result-meta">
                  {r.author && (
                    <a className="w-search-result-author" href={r.author.url}>
                      {r.author.name}
                    </a>
                  )}
                  {r.createdAt && (
                    <span className="w-search-result-date" title={preciseTime(r.createdAt)}>
                      {t('search.created', { when: relTime(r.createdAt) })}
                    </span>
                  )}
                  {r.updatedAt && (
                    <span className="w-search-result-date" title={preciseTime(r.updatedAt)}>
                      {t('search.updated', { when: relTime(r.updatedAt) })}
                    </span>
                  )}
                  {r.rating && <span className="w-search-result-rating">{r.rating}</span>}
                  <span className="w-search-result-comments">{t('search.comment-count', { count: r.comments ?? 0 })}</span>
                  {!!r.tags.length && <span className="w-search-result-tags">{r.tags.join(' ')}</span>}
                </div>
              </div>
            ))}
          </div>
          {hasMore && (
            <button className="w-search-more" onClick={onLoadMore} disabled={loadingMore}>
              {loadingMore ? t('search.loading-more') : t('search.load-more')}
            </button>
          )}
        </>
      )}

      <div className="w-search-footer">
        <Trans
          id="search.powered-by"
          children={{ icon: <img className="w-search-footer-icon" src="/-/static/images/wikitHana.png" alt="Wikit" /> }}
        />
      </div>
    </div>
  )
}

export default SearchModule
