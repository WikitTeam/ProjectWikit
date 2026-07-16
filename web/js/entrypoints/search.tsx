import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { searchModule, SearchResultItem } from '../api/search-module'
import { highlightWords } from '../reactive/pages/search/Search.utils'
import useConstCallback from '../util/const-callback'

interface Props {
  placeholder?: string
  tags?: string
  category?: string
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
  if (diff >= year) return Math.floor(diff / year) + '年前'
  if (diff >= month) return Math.floor(diff / month) + '个月前'
  if (diff >= day) return Math.floor(diff / day) + '天前'
  if (diff >= hour) return Math.floor(diff / hour) + '小时前'
  if (diff >= min) return Math.floor(diff / min) + '分钟前'
  return '刚刚'
}

function preciseTime(s: string | null): string {
  const d = parseTime(s)
  return d ? d.toLocaleString() : ''
}

const SearchModule: React.FC<Props> = ({ placeholder, tags: defaultTags }) => {
  const [q, setQ] = useState('')
  const [author, setAuthor] = useState('')
  const [tags, setTags] = useState(defaultTags || '')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

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
      setError(e?.message || '搜索失败，请稍后再试。')
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
          placeholder={placeholder || '搜索…'}
          onChange={e => setQ(e.target.value)}
        />
      </div>

      <div className="w-search-filters">
        <input className="w-search-filter" type="text" value={author} placeholder="作者" onChange={e => setAuthor(e.target.value)} />
        <input className="w-search-filter" type="text" value={tags} placeholder="标签（空格分隔，- 排除）" onChange={e => setTags(e.target.value)} />
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
        <div className="w-search-empty">未找到匹配的文章。</div>
      ) : (
        <>
          {!!results.length && <div className="w-search-total">共 {total} 条结果</div>}
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
                      创建 {relTime(r.createdAt)}
                    </span>
                  )}
                  {r.updatedAt && (
                    <span className="w-search-result-date" title={preciseTime(r.updatedAt)}>
                      更新 {relTime(r.updatedAt)}
                    </span>
                  )}
                  {r.rating && <span className="w-search-result-rating">{r.rating}</span>}
                  <span className="w-search-result-comments">{r.comments ?? 0} 条评论</span>
                  {!!r.tags.length && <span className="w-search-result-tags">{r.tags.join(' ')}</span>}
                </div>
              </div>
            ))}
          </div>
          {hasMore && (
            <button className="w-search-more" onClick={onLoadMore} disabled={loadingMore}>
              {loadingMore ? '加载中…' : '加载更多'}
            </button>
          )}
        </>
      )}

      <div className="w-search-footer">
        由 <img className="w-search-footer-icon" src="/-/static/images/wikitHana.png" alt="Wikit" /> Wikit Search 支持搜索服务
      </div>
    </div>
  )
}

export default SearchModule
