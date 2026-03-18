import * as React from 'react'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { ArticleUpdateRequest, deleteArticle, fetchArticle, updateArticle } from '../api/articles'
import sleep from '../util/async-sleep'
import useConstCallback from '../util/const-callback'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  pageId: string
  onClose?: () => void
  canDelete?: boolean
  canRename?: boolean
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

const ArticleDelete: React.FC<Props> = ({ pageId, onClose, canDelete, canRename }) => {
  const [permanent, setPermanent] = useState(false)
  const [newName, setNewName] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingSuccess, setSavingSuccess] = useState(false)
  const [error, setError] = useState('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchArticle(pageId)
      .then(data => {
        setNewName('deleted:' + data.pageId)
        setPermanent(Boolean(canDelete && !canRename))
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

    try {
      let actualNewName = newName
      if (!permanent) {
        const input: ArticleUpdateRequest = {
          pageId: newName,
          tags: [],
          forcePageId: true,
        }
        const result = await updateArticle(pageId, input)
        actualNewName = result.pageId
      } else {
        await deleteArticle(pageId)
      }
      setSaving(false)
      setSavingSuccess(true)
      await sleep(1000)
      setSavingSuccess(false)
      window.scrollTo(window.scrollX, 0)
      if (!permanent) {
        window.location.href = `/${actualNewName}`
      } else {
        window.location.reload()
      }
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
      case 'permanent':
        if (canRename) setPermanent(!permanent)
        break
    }
  })

  const onCloseError = useConstCallback(() => {
    setError('')
    if (fatalError) {
      onCancel(null)
    }
  })

  const isAlreadyDeleted = pageId.toLowerCase().startsWith('deleted:')

  if (isAlreadyDeleted && !canDelete) {
    return (
      <Styles>
        <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
          关闭
        </a>
        <h1>Удалить страницу</h1>
        <p>此页面已被标记为删除，无法再次删除</p>
      </Styles>
    )
  }

  return (
    <Styles>
      {saving && (
        <WikidotModal isLoading>
          <p>删除中...</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>删除成功!</p>
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
      <h1>删除页面</h1>
      {canDelete ? (
        <p>您可以将页面移至“deleted”分类，或永久删除（此操作不可恢复，请谨慎操作）。</p>
      ) : (
        <p>您可以将页面移至“deleted”分类以完成删除。永久删除功能不可用。</p>
      )}

      {canDelete && (
        <table className="form">
          <tbody>
            <tr>
              <td>如何操作?</td>
              <td>
                <input
                  type="checkbox"
                  name="permanent"
                  className={`text ${loading ? 'loading' : ''}`}
                  onChange={onChange}
                  id="page-rename-input"
                  checked={!permanent}
                  disabled={loading || saving || !canRename}
                />
                <label htmlFor="page-rename-input">重命名{!canRename && ' (不可用)'}</label>
              </td>
            </tr>
            <tr>
              <td></td>
              <td>
                <input
                  type="checkbox"
                  name="permanent"
                  className={`text ${loading ? 'loading' : ''}`}
                  onChange={onChange}
                  id="page-permanent-input"
                  checked={permanent}
                  disabled={loading || saving}
                />
                <label htmlFor="page-permanent-input">永久删除</label>
              </td>
            </tr>
          </tbody>
        </table>
      )}

      {!permanent ? (
        <form method="POST" onSubmit={onSubmit}>
          <p>
            为页面添加前缀“deleted:”可将其移至其他分类（命名空间）。此操作相当于删除，但信息不会丢失。
          </p>
          {isAlreadyDeleted && (
            <p>
              <strong>注意:</strong> 该页面已在“deleted”分类中。如需永久删除，请使用“永久删除”。
            </p>
          )}
          <div className="buttons form-actions">
            <input type="button" className="btn btn-danger" value="取消" onClick={onCancel} />
            {!isAlreadyDeleted && <input type="button" className="btn btn-primary" value={'移动至分类 "deleted"'} onClick={onSubmit} />}
          </div>
        </form>
      ) : (
        <form method="POST" onSubmit={onSubmit}>
          <p>此操作将永久删除页面且无法恢复。确定要继续吗？</p>
          <div className="buttons form-actions">
            <input type="button" className="btn btn-danger" value="取消" onClick={onCancel} />
            <input type="button" className="btn btn-primary" value="删除" onClick={onSubmit} />
          </div>
        </form>
      )}
    </Styles>
  )
}

export default ArticleDelete
