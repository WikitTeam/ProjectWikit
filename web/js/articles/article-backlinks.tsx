import * as React from 'react'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { ArticleBacklinks, fetchArticleBacklinks } from '../api/articles'
import useConstCallback from '../util/const-callback'
import Loader from '../util/loader'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  pageId: string
  onClose?: () => void
}

const Styles = styled.div`
  .text {
    &.loading {
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
  }
`

const ArticleBacklinksView: React.FC<Props> = ({ pageId, onClose }) => {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<ArticleBacklinks | null>(null)
  const [error, setError] = useState<string>('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchArticleBacklinks(pageId)
      .then(data => {
        setData(data)
      })
      .catch(e => {
        setFatalError(true)
        setError(e.error || 'Ошибка связи с сервером')
      })
      .finally(() => {
        setLoading(false)
      })
  }, [])

  const onCancel = useConstCallback(e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    if (onClose) onClose()
  })

  const onCloseError = useConstCallback(() => {
    setError('')
    if (fatalError) {
      onCancel(null)
    }
  })

  return (
    <Styles>
      {error && (
        <WikidotModal buttons={[{ title: '关闭', onClick: onCloseError }]} isError>
          <p>
            <strong>错误:</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        Закрыть
      </a>
      <h1>依赖此页面的其他页面</h1>
      {loading && <Loader className="loader" />}
      {data?.links?.length ? (
        <>
          <h2>反向链接</h2>
          <ul>
            {data.links.map((x, i) => (
              <li key={i}>
                <a href={`/${x.id}`} className={x.exists ? '' : 'newpage'}>
                  {x.title || x.id} ({x.id})
                </a>
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {data?.includes?.length ? (
        <>
          <h2>嵌入 (使用 [[include]])</h2>
          <ul>
            {data.includes.map((x, i) => (
              <li key={i}>
                <a href={`/${x.id}`} className={x.exists ? '' : 'newpage'}>
                  {x.title || x.id} ({x.id})
                </a>
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {data?.children?.length ? (
        <>
          <h2>子页面</h2>
          <ul>
            {data.children.map((x, i) => (
              <li key={i}>
                <a href={`/${x.id}`} className={x.exists ? '' : 'newpage'}>
                  {x.title || x.id} ({x.id})
                </a>
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {!data?.children?.length && !data?.links?.length && !data?.includes?.length && !loading && <p>此页面没有反向链接</p>}
    </Styles>
  )
}

export default ArticleBacklinksView
