import * as React from 'react'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { fetchArticle, updateArticle } from '../api/articles'
import sleep from '../util/async-sleep'
import useConstCallback from '../util/const-callback'
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

const ArticleParent: React.FC<Props> = ({ pageId, onClose }) => {
  const [parent, setParent] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingSuccess, setSavingSuccess] = useState(false)
  const [error, setError] = useState('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchArticle(pageId)
      .then(data => {
        setParent(data.parent ?? '')
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
      parent: parent,
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

  const onCancel = useConstCallback(e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    if (onClose) onClose()
  })

  const onChange = useConstCallback(e => {
    switch (e.target.name) {
      case 'parent':
        setParent(e.target.value)
        break
    }
  })

  const onClear = useConstCallback(e => {
    setParent('')
  })

  const onCloseError = useConstCallback(() => {
    setError('')
    if (fatalError) {
      onCancel(null)
    }
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
        <WikidotModal buttons={[{ title: 'Закрыть', onClick: onCloseError }]} isError>
          <p>
            <strong>错误:</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        Закрыть
      </a>
      <h1>父页面与面包屑</h1>
      <p>
        想要创建清晰的“返回”导航路径或构建站点结构？请为此页面设置父页面（上一级页面）。
      </p>
      <p>
        如果不需要{' '}
        <a
          href="https://zh.wikipedia.org/wiki/面包屑导航"
          target="_blank"
        >
          面包屑导航
        </a>{' '}
        请将输入框留空。
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>父页面名称:</td>
              <td>
                <input
                  type="text"
                  name="parent"
                  className={`text ${loading ? 'loading' : ''}`}
                  onChange={onChange}
                  id="page-parent-input"
                  defaultValue={parent}
                  disabled={loading || saving}
                />
              </td>
            </tr>
          </tbody>
        </table>
        <div className="buttons form-actions">
          <input type="button" className="btn btn-danger" value="关闭" onClick={onCancel} />
          <input type="button" className="btn btn-default" value="清除" onClick={onClear} />
          <input type="button" className="btn btn-primary" value="设置父页面" onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleParent
