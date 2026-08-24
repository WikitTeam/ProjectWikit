import { t } from '~util/i18n'
import * as React from 'react'
import Trans from '~util/trans'
import { useEffect, useState } from 'react'
import { sprintf } from 'sprintf-js'
import styled from 'styled-components'
import { ArticleLogEntry, fetchArticleLog, fetchArticleVersion } from '../api/articles'
import useConstCallback from '../util/const-callback'
import formatDate from '../util/date-format'
import Loader from '../util/loader'
import Pagination from '../util/pagination'
import UserView from '../util/user-view'
import { showVersionMessage } from '../util/wikidot-message'
import WikidotModal, { showRevertModal } from '../util/wikidot-modal'
import ArticleDiffView from './article-diff'
import ArticleSource from './article-source'

interface Props {
  pageId: string
  pathParams?: { [key: string]: string }
  onClose: () => void
}

const Styles = styled.div<{ loading?: boolean }>`
  #revision-list.loading {
    position: relative;
    min-height: calc(32px + 16px + 16px);
    &::after {
      content: ' ';
      position: absolute;
      background: #0000003f;
      z-index: 0;
      left: 0;
      right: 0;
      top: 0;
      bottom: 0;
    }
    .loader {
      position: absolute;
      left: 16px;
      top: 16px;
      z-index: 1;
    }
  }
  .page-history {
    tr td {
      &:nth-child(2) {
        width: 5em;
      }
      &:nth-child(4) {
        width: 5em;
      }
      &:nth-child(5) {
        width: 15em;
      }
      &:nth-child(6) {
        padding: 0 0.5em;
        width: 12em;
      }
      &:nth-child(7) {
        font-size: 90%;
      }
      .action {
        border: 1px solid #bbb;
        padding: 0 3px;
        text-decoration: none;
        color: #824;
        background: transparent;
        cursor: pointer;
      }
    }
  }
`

export function renderArticleHistoryFlags(entry: ArticleLogEntry) {
  const renderType = (type: string) => {
    switch (type) {
      case 'new':
        return (
          <span className="spantip" title={t('articles.history.flag-new')}>
            N
          </span>
        )

      case 'title':
        return (
          <span className="spantip" title={t('articles.history.flag-title')}>
            T
          </span>
        )

      case 'source':
        return (
          <span className="spantip" title={t('articles.history.flag-source')}>
            S
          </span>
        )

      case 'tags':
        return (
          <span className="spantip" title={t('articles.history.flag-tags')}>
            A
          </span>
        )

      case 'name':
        return (
          <span className="spantip" title={t('articles.history.flag-name')}>
            R
          </span>
        )

      case 'parent':
        return (
          <span className="spantip" title={t('articles.history.flag-parent')}>
            M
          </span>
        )

      case 'file_added':
        return (
          <span className="spantip" title={t('articles.history.flag-file-added')}>
            F
          </span>
        )

      case 'file_deleted':
        return (
          <span className="spantip" title={t('articles.history.flag-file-deleted')}>
            F
          </span>
        )

      case 'file_renamed':
        return (
          <span className="spantip" title={t('articles.history.flag-file-renamed')}>
            F
          </span>
        )

      case 'votes_deleted':
        return (
          <span className="spantip" title={t('articles.history.flag-votes')}>
            V
          </span>
        )

      case 'wikidot':
        return (
          <span className="spantip" title={t('articles.history.flag-migrated')}>
            W
          </span>
        )
    }
  }

  if (entry.meta.subtypes) {
    return entry.meta.subtypes.map((x: any) => <React.Fragment key={x}>{renderType(x)}</React.Fragment>)
  } else {
    return renderType(entry.type)
  }
}

export function renderArticleHistoryComment(entry: ArticleLogEntry) {
  if (entry.comment.trim()) {
    return entry.comment
  }
  return entry.defaultComment

  switch (entry.type) {
    case 'new':
      return t('articles.history.created')

    case 'title':
      return (
        <Trans
          id="articles.history.title-changed"
          children={{ from: <em>{entry.meta.prev_title}</em>, to: <em>{entry.meta.title}</em> }}
        />
      )

    case 'name':
      return (
        <Trans
          id="articles.history.name-changed"
          children={{ from: <em>{entry.meta.prev_name}</em>, to: <em>{entry.meta.name}</em> }}
        />
      )

    case 'tags':
      let added_tags = entry.meta.added_tags.map((tag: any) => tag['name'])
      let removed_tags = entry.meta.removed_tags.map((tag: any) => tag['name'])
      if (Array.isArray(added_tags) && added_tags.length && Array.isArray(removed_tags) && removed_tags.length) {
        return (
          <>{t('articles.history.tags-added-removed', { added: added_tags.join(', '), removed: removed_tags.join(', ') })}</>
        )
      } else if (Array.isArray(added_tags) && added_tags.length) {
        return <>{t('articles.history.tags-added', { added: added_tags.join(', ') })}</>
      } else if (Array.isArray(removed_tags) && removed_tags.length) {
        return <>{t('articles.history.tags-removed', { removed: removed_tags.join(', ') })}</>
      }
      break

    case 'parent':
      if (entry.meta.prev_parent && entry.meta.parent) {
        return (
          <Trans
            id="articles.history.parent-changed"
            children={{ from: <em>{entry.meta.prev_parent}</em>, to: <em>{entry.meta.parent}</em> }}
          />
        )
      } else if (entry.meta.prev_parent) {
        return (
          <Trans id="articles.history.parent-removed" children={{ parent: <em>{entry.meta.prev_parent}</em> }} />
        )
      } else if (entry.meta.parent) {
        return (
          <Trans id="articles.history.parent-set" children={{ parent: <em>{entry.meta.parent}</em> }} />
        )
      }
      break

    case 'file_added':
      return (
        <Trans id="articles.history.file-added" children={{ name: <em>{entry.meta.name}</em> }} />
      )

    case 'file_deleted':
      return (
        <Trans id="articles.history.file-deleted" children={{ name: <em>{entry.meta.name}</em> }} />
      )

    case 'file_renamed':
      return (
        <Trans
          id="articles.history.file-renamed"
          children={{ from: <em>{entry.meta.prev_name}</em>, to: <em>{entry.meta.name}</em> }}
        />
      )

    case 'votes_deleted': {
      let ratingStr = 'n/a'
      if (entry.meta.rating_mode === 'updown') {
        ratingStr = sprintf('%+d', entry.meta.rating)
      } else if (entry.meta.rating_mode === 'stars') {
        ratingStr = sprintf('%.1f', entry.meta.rating)
      }
      return (
        <>
          {t('articles.history.votes-deleted', {
            rating: ratingStr,
            votes: entry.meta.votes_count,
            popularity: entry.meta.popularity,
          })}
        </>
      )
    }

    case 'authorship': {
      let added_authors = entry.meta.added_authors
      let removed_authors = entry.meta.removed_authors
      if (Array.isArray(added_authors) && added_authors.length && Array.isArray(removed_authors) && removed_authors.length) {
        return (
          <>{t('articles.history.authors-added-removed', { added: added_authors.join(', '), removed: removed_authors.join(', ') })}</>
        )
      } else if (Array.isArray(added_authors) && added_authors.length) {
        return <>{t('articles.history.authors-added', { added: added_authors.join(', ') })}</>
      } else if (Array.isArray(removed_authors) && removed_authors.length) {
        return <>{t('articles.history.authors-removed', { removed: removed_authors.join(', ') })}</>
      }
    }

    case 'revert':
      return <>{t('articles.history.reverted', { revision: entry.meta.rev_number })}</>
  }
}

const ArticleHistory: React.FC<Props> = ({ pageId, pathParams, onClose: onCloseDelegate }) => {
  const [loading, setLoading] = useState(false)
  const [entries, setEntries] = useState<Array<ArticleLogEntry>>([])
  const [subarea, setSubarea] = useState<React.ReactNode>()
  const [entryCount, setEntryCount] = useState(0)
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(25)
  const [error, setError] = useState('')
  const [fatalError, setFatalError] = useState(false)
  const [firstCompareEntry, setFirstCompareEntry] = useState<ArticleLogEntry>()
  const [secondCompareEntry, setSecondCompareEntry] = useState<ArticleLogEntry>()

  useEffect(() => {
    loadHistory()
  }, [])

  const loadHistory = useConstCallback(async (nextPage?: number) => {
    setLoading(true)
    setError('')

    const realPage = nextPage || page
    const from = (realPage - 1) * perPage
    const to = realPage * perPage

    fetchArticleLog(pageId, from, to)
      .then(history => {
        setEntries(history.entries)
        setEntryCount(history.count)
        setPage(realPage)
        setFirstCompareEntry(history.entries[1])
        setSecondCompareEntry(history.entries[0])
      })
      .catch(e => {
        setFatalError(entries === null)
        setError(e.error || t('common.server-unreachable'))
      })
      .finally(() => {
        setLoading(false)
      })
  })

  const onClose = useConstCallback(e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    if (onCloseDelegate) onCloseDelegate()
  })

  const onCloseError = useConstCallback(() => {
    setError('')
    if (fatalError) {
      onClose(null)
    }
  })

  const onChangePage = useConstCallback(nextPage => {
    loadHistory(nextPage)
  })

  const renderActions = useConstCallback((entry: ArticleLogEntry) => {
    if (entry.type === 'wikidot') {
      return null
    }
    return (
      <>
        <a href="#" onClick={e => displayArticleVersion(e, entry)} title={t('articles.history.view-version')}>
          V
        </a>
        <a href="#" onClick={e => displayVersionSource(e, entry)} title={t('articles.history.view-source')}>
          S
        </a>
        {entryCount !== entry.revNumber + 1 && (
          <a href="#" onClick={e => revertArticleVersion(e, entry)} title={t('articles.history.revert')}>
            R
          </a>
        )}
      </>
    )
  })

  const renderUser = useConstCallback((entry: ArticleLogEntry) => {
    return <UserView data={entry.user} />
  })

  const renderDate = useConstCallback((entry: ArticleLogEntry) => {
    return formatDate(new Date(entry.createdAt))
  })

  const displayArticleVersion = useConstCallback((e: React.MouseEvent, entry: ArticleLogEntry) => {
    e.preventDefault()
    e.stopPropagation()

    fetchArticleVersion(pageId, entry.revNumber, pathParams).then(function (resp) {
      showVersionMessage(entry.revNumber, new Date(entry.createdAt), entry.user, pageId)
      document.getElementById('page-content')!.innerHTML = resp.rendered
    })
  })

  const displayVersionSource = useConstCallback((e: React.MouseEvent, entry: ArticleLogEntry) => {
    e.preventDefault()
    e.stopPropagation()

    fetchArticleVersion(pageId, entry.revNumber, pathParams).then(function (resp) {
      hideSubArea()
      showSubArea(<ArticleSource pageId={pageId} onClose={hideSubArea} source={resp.source} />)
    })
  })

  const displayVersionDiff = useConstCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()

    if (firstCompareEntry && secondCompareEntry) {
      hideSubArea()
      showSubArea(
        <ArticleDiffView
          pageId={pageId}
          onClose={hideSubArea}
          firstEntry={firstCompareEntry}
          secondEntry={secondCompareEntry}
          pathParams={pathParams}
        />,
      )
    }
  })

  const showSubArea = useConstCallback((component: React.ReactNode) => {
    setSubarea(component)
  })

  const hideSubArea = useConstCallback(() => {
    setSubarea(undefined)
  })

  const revertArticleVersion = useConstCallback((e: React.MouseEvent, entry: ArticleLogEntry) => {
    e.preventDefault()
    e.stopPropagation()

    showRevertModal(pageId, entry)
  })

  const totalPages = Math.ceil(entryCount / perPage)

  return (
    <Styles>
      {error && (
        <WikidotModal buttons={[{ title: t('articles.history.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.history.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onClose}>
        {t('articles.history.close')}
      </a>
      <h1>{t('articles.history.title')}</h1>
      <div id="revision-list" className={`${loading ? 'loading' : ''}`}>
        {loading && <Loader className="loader" />}
        <div className="buttons">
          <input type="button" className="btn btn-default btn-sm" value={t('articles.history.refresh')} onClick={() => loadHistory()} />
          <input
            type="button"
            className="btn btn-default btn-sm"
            value={t('articles.history.compare')}
            name="compare"
            id="history-compare-button"
            onClick={displayVersionDiff}
          />
        </div>
        {entries && totalPages > 1 && <Pagination page={page} maxPages={totalPages} onChange={onChangePage} />}
        {entries && (
          <table className="page-history">
            <tbody>
              <tr>
                <td>{t('articles.history.column-revision')}</td>
                <td>&nbsp;</td>
                <td>{t('articles.history.column-flags')}</td>
                <td>{t('articles.history.column-actions')}</td>
                <td>{t('articles.history.column-author')}</td>
                <td>{t('articles.history.column-date')}</td>
                <td>{t('articles.history.column-comment')}</td>
              </tr>
              {entries.map(entry => {
                return (
                  <tr key={entry.revNumber} id={`revision-row-${entry.revNumber}`}>
                    {/* BHL has CSS selector that says tr[id*="evision-row"] */}
                    <td>{entry.revNumber}.</td>
                    <td style={{ width: '5em' }}>
                      <input
                        type="radio"
                        name="from"
                        value={entry.revNumber}
                        onChange={() => {
                          setFirstCompareEntry(entry)
                        }}
                        defaultChecked={entries[1] === entry}
                      />
                      <input
                        type="radio"
                        name="to"
                        value={entry.revNumber}
                        onChange={() => {
                          setSecondCompareEntry(entry)
                        }}
                        defaultChecked={entries[0] === entry}
                      />
                    </td>
                    <td>{renderArticleHistoryFlags(entry)}</td>
                    <td className="optionstd" style={{ width: '5em' }}>
                      {renderActions(entry)}
                    </td>
                    <td style={{ width: '15em' }}>{renderUser(entry)}</td>
                    <td style={{ padding: '0 0.5em', width: '7em' }}>{renderDate(entry)}</td>
                    <td style={{ fontSize: '90%' }}>{renderArticleHistoryComment(entry)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>
      <div id="history-subarea">{subarea}</div>
    </Styles>
  )
}

export default ArticleHistory
