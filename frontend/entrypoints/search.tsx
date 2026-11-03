import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { highlightWords } from '~util/highlight-words'
import { t } from '~util/i18n'
import Loader from '~util/loader'
import { ForumPlace, forumPlaces, ForumResultItem, searchForum, searchModule, SearchResultItem } from '../api/search-module'
import useConstCallback from '../util/const-callback'
import formatDate from '../util/date-format'
import SearchSuggestInput from './search-suggest'

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
  return d ? formatDate(d) : ''
}

const SearchModule: React.FC<Props> = ({
  placeholder,
  tags: defaultTags,
  category: defaultCategory,
  q: initialQ,
  author: initialAuthor,
  datefrom: initialDateFrom,
  dateto: initialDateTo,
}) => {
  const [q, setQ] = useState(initialQ || '')
  const [author, setAuthor] = useState(initialAuthor || '')
  const [tags, setTags] = useState(defaultTags || '')
  const [category, setCategory] = useState(defaultCategory || '')
  const [dateFrom, setDateFrom] = useState(initialDateFrom || '')
  const [dateTo, setDateTo] = useState(initialDateTo || '')

  const [scope, setScope] = useState<'pages' | 'forum'>('pages')
  const [place, setPlace] = useState('')
  const [places, setPlaces] = useState<ForumPlace[] | null>(null)
  const [results, setResults] = useState<SearchResultItem[]>([])
  const [posts, setPosts] = useState<ForumResultItem[]>([])
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
      if (scope === 'forum') {
        const found = await searchForum({ q: q.trim(), author: author.trim(), category: place, datefrom: dateFrom, dateto: dateTo, offset })
        if (my !== seq.current) return
        setPosts(prev => (append ? [...prev, ...found.results] : found.results))
        setHasMore(found.hasMore)
        setSearched(true)
        return
      }
      const resp = await searchModule({
        q: q.trim(),
        author: author.trim(),
        tags: tags.trim(),
        category: category.trim(),
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
    const shared = !!(q.trim() || author.trim() || dateFrom || dateTo)
    const hasCriteria = scope === 'forum' ? shared || !!place : shared || !!(tags.trim() || category.trim())
    if (!hasCriteria) {
      seq.current++
      setResults([])
      setPosts([])
      setHasMore(false)
      setTotal(0)
      setSearched(false)
      setError(null)
      return
    }
    const t = setTimeout(() => runSearch(0, false), 300)
    return () => clearTimeout(t)
  }, [scope, q, author, tags, category, place, dateFrom, dateTo])

  useEffect(() => {
    if (scope !== 'forum' || places) return
    forumPlaces()
      .then(setPlaces)
      .catch(() => setPlaces([]))
  }, [scope])

  const groups = (places || []).reduce<[string, ForumPlace[]][]>((acc, one) => {
    const last = acc[acc.length - 1]
    if (last && last[0] === one.group) last[1].push(one)
    else acc.push([one.group, [one]])
    return acc
  }, [])

  const onLoadMore = useConstCallback(() => runSearch(scope === 'forum' ? posts.length : results.length, true))

  const pick = (next: 'pages' | 'forum') => () => {
    if (next === scope) return
    setResults([])
    setPosts([])
    setSearched(false)
    setScope(next)
  }
  const shown = scope === 'forum' ? posts.length : results.length

  return (
    <div className="w-search">
      <div className="w-search-switch" role="tablist" aria-label={t('search.scope')}>
        {(['pages', 'forum'] as const).map(one => (
          <button key={one} type="button" role="tab" className="w-search-switch-item" aria-selected={scope === one} onClick={pick(one)}>
            {t(one === 'pages' ? 'search.scope-pages' : 'search.scope-forum')}
          </button>
        ))}
      </div>
      <div className="w-search-box">
        <input
          className="w-search-input"
          type="text"
          value={q}
          placeholder={scope === 'forum' ? t('search.placeholder-forum') : placeholder || t('search.placeholder')}
          onChange={e => setQ(e.target.value)}
        />
      </div>

      <div className="w-search-filters">
        <SearchSuggestInput
          field="author"
          value={author}
          placeholder={t(scope === 'forum' ? 'search.poster-placeholder' : 'search.author-placeholder')}
          onChange={setAuthor}
        />
        {scope === 'pages' && (
          <>
            <SearchSuggestInput field="tags" value={tags} placeholder={t('search.tags-placeholder')} onChange={setTags} />
            <SearchSuggestInput field="category" value={category} placeholder={t('search.category-placeholder')} onChange={setCategory} />
          </>
        )}
        {scope === 'forum' && (
          <select
            className={`w-search-filter w-search-place${place ? '' : ' is-empty'}`}
            value={place}
            aria-label={t('search.place-placeholder')}
            onChange={e => setPlace(e.target.value)}
          >
            <option value="">{t('search.place-placeholder')}</option>
            {groups.map(([group, items]) =>
              group ? (
                <optgroup key={group} label={group}>
                  {items.map(one => (
                    <option key={one.value} value={one.value}>
                      {one.name}
                    </option>
                  ))}
                </optgroup>
              ) : (
                items.map(one => (
                  <option key={one.value} value={one.value}>
                    {one.name}
                  </option>
                ))
              ),
            )}
          </select>
        )}
        <div className="w-search-dates">
          <input className="w-search-filter w-search-date" type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)} />
          <span className="w-search-date-sep">–</span>
          <input className="w-search-filter w-search-date" type="date" value={dateTo} onChange={e => setDateTo(e.target.value)} />
        </div>
      </div>

      {error && <div className="w-search-error">{error}</div>}

      {loading ? (
        <Loader className="w-search-loader" />
      ) : searched && !shown && !error ? (
        <div className="w-search-empty">{t(scope === 'forum' ? 'search.empty-forum' : 'search.empty')}</div>
      ) : (
        <>
          {scope === 'pages' && !!results.length && <div className="w-search-total">{t('search.total', { total })}</div>}
          {scope === 'forum' && (
            <ol className="w-search-posts">
              {posts.map((r, i) => (
                <li className="w-search-post" key={`${r.url}-${i}`}>
                  <div className="w-search-post-head">
                    {r.where && <span className="w-search-post-where">{r.where}</span>}
                    <a className="w-search-post-thread" href={r.url}>
                      {highlightWords(r.title, r.words)}
                    </a>
                  </div>
                  <div className="w-search-post-body">
                    {r.post && r.post !== r.title && <strong className="w-search-post-name">{highlightWords(r.post, r.words)}</strong>}
                    {highlightWords(r.excerpt, r.words)}
                  </div>
                  <div className="w-search-post-by">
                    {r.author && (
                      <a className="w-search-result-author" href={r.author.url}>
                        {r.author.name}
                      </a>
                    )}
                    {r.createdAt && (
                      <span className="w-search-result-date" title={preciseTime(r.createdAt)}>
                        {t('search.posted', { when: relTime(r.createdAt) })}
                      </span>
                    )}
                  </div>
                </li>
              ))}
            </ol>
          )}
          {scope === 'pages' && (
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
          )}
          {loadingMore && <Loader className="w-search-loader" />}
          {hasMore && !loadingMore && (
            <button className="w-search-more" onClick={onLoadMore}>
              {t('search.load-more')}
            </button>
          )}
        </>
      )}
    </div>
  )
}

export default SearchModule
