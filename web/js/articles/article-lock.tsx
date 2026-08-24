import { t } from '~util/i18n'
import * as React from 'react'
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

const ArticleLock: React.FC<Props> = ({ pageId, isNew, onClose }) => {
  const [locked, setLocked] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingSuccess, setSavingSuccess] = useState(false)
  const [error, setError] = useState('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    fetchArticle(pageId)
      .then(data => {
        setLocked(Boolean(data.locked))
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
      pageId: pageId,
      locked: locked,
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
    let value = e.target.value
    if (e.target.type === 'checkbox') value = e.target.checked

    switch (e.target.name) {
      case 'locked':
        setLocked(value)
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
          <p>{t('articles.lock.saving')}</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>{t('articles.lock.saved')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('articles.lock.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.lock.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.lock.close')}
      </a>
      <h1>{t('articles.lock.title')}</h1>
      <p>
        {t('articles.lock.note')}
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.lock.locked-label')}</td>
              <td>
                <input
                  type="checkbox"
                  name="locked"
                  className={`text ${loading ? 'loading' : ''}`}
                  onChange={onChange}
                  id="page-locked-input"
                  checked={locked}
                  disabled={loading || saving}
                />
              </td>
            </tr>
          </tbody>
        </table>
        <div className="buttons form-actions">
          <input type="button" className="btn btn-danger" value={t('articles.lock.cancel')} onClick={onCancel} />
          <input type="button" className="btn btn-primary" value={t('articles.lock.save')} onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleLock
