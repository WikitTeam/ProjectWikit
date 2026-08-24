import { t } from '~util/i18n'
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
        setError(e.error || t('common.server-unreachable'))
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
      setError(e.error || t('common.server-unreachable'))
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
          {t('articles.delete.close-blocked')}
        </a>
        <h1>{t('articles.delete.title-blocked')}</h1>
        <p>{t('articles.delete.already-deleted')}</p>
      </Styles>
    )
  }

  return (
    <Styles>
      {saving && (
        <WikidotModal isLoading>
          <p>{t('articles.delete.deleting')}</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>{t('articles.delete.deleted')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('articles.delete.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.delete.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.delete.close')}
      </a>
      <h1>{t('articles.delete.title')}</h1>
      {canDelete ? (
        <p>{t('articles.delete.note')}</p>
      ) : (
        <p>{t('articles.delete.note-no-permanent')}</p>
      )}

      {canDelete && (
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.delete.how-label')}</td>
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
                <label htmlFor="page-rename-input">{t('articles.delete.option-rename')}{!canRename && t('articles.delete.option-unavailable')}</label>
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
                <label htmlFor="page-permanent-input">{t('articles.delete.option-permanent')}</label>
              </td>
            </tr>
          </tbody>
        </table>
      )}

      {!permanent ? (
        <form method="POST" onSubmit={onSubmit}>
          <p>
            {t('articles.delete.rename-note')}
          </p>
          {isAlreadyDeleted && (
            <p>
              <strong>{t('articles.delete.warning-label')}</strong> {t('articles.delete.warning-already-deleted')}
            </p>
          )}
          <div className="buttons form-actions">
            <input type="button" className="btn btn-danger" value={t('articles.delete.cancel')} onClick={onCancel} />
            {!isAlreadyDeleted && <input type="button" className="btn btn-primary" value={t('articles.delete.move')} onClick={onSubmit} />}
          </div>
        </form>
      ) : (
        <form method="POST" onSubmit={onSubmit}>
          <p>{t('articles.delete.confirm-permanent')}</p>
          <div className="buttons form-actions">
            <input type="button" className="btn btn-danger" value={t('articles.delete.confirm-cancel')} onClick={onCancel} />
            <input type="button" className="btn btn-primary" value={t('articles.delete.confirm')} onClick={onSubmit} />
          </div>
        </form>
      )}
    </Styles>
  )
}

export default ArticleDelete
