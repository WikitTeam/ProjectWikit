import { t } from '~util/i18n'
import * as React from 'react'
import Trans from '~util/trans'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { fetchArticle, updateArticle } from '../api/articles'
import sleep from '../util/async-sleep'
import useConstCallback from '../util/const-callback'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  pageId: string
  isNew?: boolean
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

const ArticleRename: React.FC<Props> = ({ pageId, isNew, onClose }) => {
  const [newName, setNewName] = useState(pageId)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingSuccess, setSavingSuccess] = useState(false)
  const [error, setError] = useState('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchArticle(pageId)
      .then(data => {
        setNewName(data.pageId)
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

    const input = {
      pageId: newName,
    }

    try {
      await updateArticle(pageId, input)
      setSavingSuccess(true)
      setSaving(false)
      await sleep(1000)
      setSavingSuccess(false)
      window.scrollTo(window.scrollX, 0)
      window.location.href = `/${newName}`
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
      case 'newName':
        setNewName(e.target.value)
        break
    }
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
          <p>{t('articles.rename.saving')}</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>{t('articles.rename.saved')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('articles.rename.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.rename.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.rename.close')}
      </a>
      <h1>{t('articles.rename.title')}</h1>
      <p>
        <Trans id="articles.rename.note" children={{ action: <em>{t('articles.rename.note-action')}</em> }} />{' '}
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.rename.current-name-label')}</td>
              <td>{pageId}</td>
            </tr>
            <tr>
              <td>{t('articles.rename.new-name-label')}</td>
              <td>
                <input
                  type="text"
                  name="newName"
                  className={`text ${loading ? 'loading' : ''}`}
                  onChange={onChange}
                  id="page-rename-input"
                  defaultValue={newName}
                  disabled={loading || saving}
                  autoFocus
                />
              </td>
            </tr>
          </tbody>
        </table>
        <div className="buttons form-actions">
          <input type="button" className="btn btn-danger" value={t('articles.rename.cancel')} onClick={onCancel} />
          <input type="button" className="btn btn-primary" value={t('articles.rename.submit')} onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleRename
