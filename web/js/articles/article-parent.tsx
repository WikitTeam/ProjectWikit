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
  onClose?: () => void
}

const Styles = styled.div`
  .text {
    &.loading {
      .loader {
        position: absolute;
        left: 0;
        right: 0;
        top: 0;
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
          <p>{t('articles.parent.saving')}</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>{t('articles.parent.saved')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('articles.parent.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.parent.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.parent.close')}
      </a>
      <h1>{t('articles.parent.title')}</h1>
      <p>
        {t('articles.parent.note')}
      </p>
      <p>
        {t('articles.parent.note-before-link')}{' '}
        <a
          href={t('articles.parent.breadcrumb-wiki-url')}
          target="_blank"
        >
          {t('articles.parent.breadcrumb-wiki-label')}
        </a>{' '}
        {t('articles.parent.note-after-link')}
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.parent.parent-name-label')}</td>
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
          <input type="button" className="btn btn-danger" value={t('articles.parent.cancel')} onClick={onCancel} />
          <input type="button" className="btn btn-default" value={t('articles.parent.clear')} onClick={onClear} />
          <input type="button" className="btn btn-primary" value={t('articles.parent.save')} onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleParent
