import * as React from 'react'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { fetchArticle, updateArticle } from '../api/articles'
import { fetchAllTags, FetchAllTagsResponse } from '../api/tags'
import TagEditorComponent from '../components/tag-editor'
import sleep from '../util/async-sleep'
import useConstCallback from '../util/const-callback'
import Loader from '../util/loader'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  pageId: string
  isNew?: boolean
  onClose?: () => void
  canCreateTags?: boolean
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

  .w-tag-editor-container {
    position: relative;
  }

  /* fixes BHL; without table this looks bad */
  table.form {
    display: table !important;
  }

  .form tr {
    display: table-row !important;
  }

  .form td,
  th {
    display: table-cell !important;
  }
`

const ArticleTags: React.FC<Props> = ({ pageId, isNew, onClose, canCreateTags }) => {
  const [tags, setTags] = useState<Array<string>>([])
  const [allTags, setAllTags] = useState<FetchAllTagsResponse>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingSuccess, setSavingSuccess] = useState(false)
  const [error, setError] = useState('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    Promise.all([fetchArticle(pageId), fetchAllTags()])
      .then(([data, allTags]) => {
        setTags(data.tags ?? [])
        setAllTags(allTags)
      })
      .catch(e => {
        setFatalError(true)
        setError(e.error || '连接服务器失败')
      })
      .finally(() => {
        setLoading(false)
      })
  }, [])

  const onSubmit = useConstCallback(async e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }

    setSaving(true)
    setError('')
    setSavingSuccess(false)

    const input = {
      pageId: pageId,
      tags: tags,
    }

    try {
      await updateArticle(pageId, input)
      setSavingSuccess(true)
      setSaving(false)
      await sleep(1000)
      setSavingSuccess(false)
      window.scrollTo(window.scrollX, 0)
      window.location.reload()
    } catch (e) {
      setFatalError(false)
      setError(e.error || '连接服务器失败')
    } finally {
      setSaving(false)
    }
  })

  const onClear = useConstCallback(e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    setTags([])
  })

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

  const onChange = useConstCallback((tags: Array<string>) => {
    setTags(tags)
  })

  return (
    <Styles>
      {saving && (
        <WikidotModal isLoading>
          <p>保存中...</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>保存成功!</p>
        </WikidotModal>
      )}
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
      <h1>页面标签</h1>
      <p>
        标签是整理站点内容、在相关页面间建立“横向导航”的有效方式。您可以为每个页面添加多个标签。了解更多关于{' '}
        <a href="http://zh.wikipedia.org/wiki/标签_(元数据)" target="_blank">
          {' '}
          标签
        </a>
        , 以及{' '}
        <a href="http://zh.wikipedia.org/wiki/标签云" target="_blank">
          标签云{' '}
        </a>
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>标签:</td>
            </tr>
            <tr>
              <td className="w-tag-editor-container">
                {loading && <Loader className="loader" />}
                <TagEditorComponent canCreateTags={canCreateTags} tags={tags} allTags={allTags || { categories: [], tags: [] }} onChange={onChange} />
              </td>
            </tr>
          </tbody>
        </table>
        <div className="buttons form-actions">
          <input type="button" className="btn btn-danger" value="关闭" onClick={onCancel} />
          <input type="button" className="btn btn-default" value="清除" onClick={onClear} />
          <input type="button" className="btn btn-primary" value="保存标签" onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleTags
