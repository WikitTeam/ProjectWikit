import * as React from 'react'
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
          <span className="spantip" title="已创建新页面">
            N
          </span>
        )

      case 'title':
        return (
          <span className="spantip" title="标题已更改">
            T
          </span>
        )

      case 'source':
        return (
          <span className="spantip" title="内容已更改">
            S
          </span>
        )

      case 'tags':
        return (
          <span className="spantip" title="标签已更改">
            A
          </span>
        )

      case 'name':
        return (
          <span className="spantip" title="页面已重命名/删除">
            R
          </span>
        )

      case 'parent':
        return (
          <span className="spantip" title="父页面已更改">
            M
          </span>
        )

      case 'file_added':
        return (
          <span className="spantip" title="已上传文件">
            F
          </span>
        )

      case 'file_deleted':
        return (
          <span className="spantip" title="已删除文件">
            F
          </span>
        )

      case 'file_renamed':
        return (
          <span className="spantip" title="文件已重命名">
            F
          </span>
        )

      case 'votes_deleted':
        return (
          <span className="spantip" title="评分已更改">
            V
          </span>
        )

      case 'wikidot':
        return (
          <span className="spantip" title="从Wikidot迁移的版本">
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
      return '正在创建新页面'

    case 'title':
      return (
        <>
          标题已从 "<em>{entry.meta.prev_title}</em>" 更改为 "<em>{entry.meta.title}</em>"
        </>
      )

    case 'name':
      return (
        <>
          页面已从 "<em>{entry.meta.prev_name}</em>" 更改为 "<em>{entry.meta.name}</em>"
        </>
      )

    case 'tags':
      let added_tags = entry.meta.added_tags.map((tag: any) => tag['name'])
      let removed_tags = entry.meta.removed_tags.map((tag: any) => tag['name'])
      if (Array.isArray(added_tags) && added_tags.length && Array.isArray(removed_tags) && removed_tags.length) {
        return (
          <>
            已添加标签: {added_tags.join(', ')}. 已移除标签: {removed_tags.join(', ')}.
          </>
        )
      } else if (Array.isArray(added_tags) && added_tags.length) {
        return <>Добавлены теги: {added_tags.join(', ')}.</>
      } else if (Array.isArray(removed_tags) && removed_tags.length) {
        return <>Удалены теги: {removed_tags.join(', ')}.</>
      }
      break

    case 'parent':
      if (entry.meta.prev_parent && entry.meta.parent) {
        return (
          <>
            父页面已从 "<em>{entry.meta.prev_parent}</em>" 更改为 "<em>{entry.meta.parent}</em>"
          </>
        )
      } else if (entry.meta.prev_parent) {
        return (
          <>
            已移除父页面 "<em>{entry.meta.prev_parent}</em>"
          </>
        )
      } else if (entry.meta.parent) {
        return (
          <>
            已设置父页面 "<em>{entry.meta.parent}</em>"
          </>
        )
      }
      break

    case 'file_added':
      return (
        <>
          已上传文件: "<em>{entry.meta.name}</em>"
        </>
      )

    case 'file_deleted':
      return (
        <>
          已删除文件: "<em>{entry.meta.name}</em>"
        </>
      )

    case 'file_renamed':
      return (
        <>
          文件已重命名: "<em>{entry.meta.prev_name}</em>" 至 "<em>{entry.meta.name}</em>"
        </>
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
          页面评分已重置: {ratingStr} (评分数: {entry.meta.votes_count}, 人气: {entry.meta.popularity}%)
        </>
      )
    }

    case 'authorship': {
      let added_authors = entry.meta.added_authors
      let removed_authors = entry.meta.removed_authors
      if (Array.isArray(added_authors) && added_authors.length && Array.isArray(removed_authors) && removed_authors.length) {
        return (
          <>
            已添加作者: {added_authors.join(', ')}. 已移除作者: {removed_authors.join(', ')}.
          </>
        )
      } else if (Array.isArray(added_authors) && added_authors.length) {
        return <>已添加作者: {added_authors.join(', ')}.</>
      } else if (Array.isArray(removed_authors) && removed_authors.length) {
        return <>已移除作者: {removed_authors.join(', ')}.</>
      }
    }

    case 'revert':
      return <>页面已回滚至版本 #{entry.meta.rev_number}</>
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
        setError(e.error || 'Ошибка связи с сервером')
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
        <a href="#" onClick={e => displayArticleVersion(e, entry)} title="检视页面版本">
          V
        </a>
        <a href="#" onClick={e => displayVersionSource(e, entry)} title="检视页面源代码">
          S
        </a>
        {entryCount !== entry.revNumber + 1 && (
          <a href="#" onClick={e => revertArticleVersion(e, entry)} title="回复至修订版本">
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
        <WikidotModal buttons={[{ title: '关闭', onClick: onCloseError }]} isError>
          <p>
            <strong>错误:</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onClose}>
        Закрыть
      </a>
      <h1>页面更改历史</h1>
      <div id="revision-list" className={`${loading ? 'loading' : ''}`}>
        {loading && <Loader className="loader" />}
        <div className="buttons">
          <input type="button" className="btn btn-default btn-sm" value="更新列表" onClick={() => loadHistory()} />
          <input
            type="button"
            className="btn btn-default btn-sm"
            value="比较版本"
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
                <td>版本</td>
                <td>&nbsp;</td>
                <td>标记</td>
                <td>操作</td>
                <td>由</td>
                <td>日期</td>
                <td>评论</td>
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
