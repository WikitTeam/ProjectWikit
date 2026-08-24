import { t } from '~util/i18n'
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
      setError(e.error || t('common.server-unreachable'))
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
          <p>{t('articles.tags.saving')}</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>{t('articles.tags.saved')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('articles.tags.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.tags.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.tags.close')}
      </a>
      <h1>{t('articles.tags.title')}</h1>
      <p>
        {t('articles.tags.note')}{' '}
        <a href={t('articles.tags.tag-wiki-url')} target="_blank">
          {' '}
          {t('articles.tags.tag-wiki-label')}
        </a>
        {t('articles.tags.note-and')}{' '}
        <a href={t('articles.tags.tag-cloud-wiki-url')} target="_blank">
          {t('articles.tags.tag-cloud-wiki-label')}{' '}
        </a>
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.tags.tags-label')}</td>
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
          <input type="button" className="btn btn-danger" value={t('articles.tags.cancel')} onClick={onCancel} />
          <input type="button" className="btn btn-default" value={t('articles.tags.clear')} onClick={onClear} />
          <input type="button" className="btn btn-primary" value={t('articles.tags.save')} onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleTags
