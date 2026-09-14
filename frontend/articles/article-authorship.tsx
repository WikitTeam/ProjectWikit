import * as React from 'react'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { fetchArticle, updateArticle } from '../api/articles'
import { fetchAllUsers, UserData } from '../api/user'
import AuthorshipEditorComponent from '../components/authorship-editor'
import sleep from '../util/async-sleep'
import useConstCallback from '../util/const-callback'
import { t } from '~util/i18n'
import Loader from '../util/loader'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  user: UserData | null
  pageId: string
  isNew?: boolean
  editable?: boolean
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

  .w-authorship-editor-container {
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

const ArticleAuthorship: React.FC<Props> = ({ user, pageId, editable, onClose }) => {
  const [originAuthors, setOriginAuthors] = useState<UserData[]>([])
  const [authors, setAuthors] = useState<UserData[]>([])
  const [allUsers, setAllUsers] = useState<UserData[]>([])
  const [askTransferOwnership, setAskTransferOwnership] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [savingSuccess, setSavingSuccess] = useState(false)
  const [error, setError] = useState<string>('')
  const [fatalError, setFatalError] = useState(false)

  useEffect(() => {
    setLoading(true)
    Promise.all([fetchArticle(pageId), fetchAllUsers()])
      .then(([data, allUsers]) => {
        setOriginAuthors(data?.authors || [])
        setAuthors(data?.authors || [])
        setAllUsers(allUsers)
      })
      .catch(e => {
        setFatalError(true)
        setError(e.error || t('common.server-unreachable'))
      })
      .finally(() => {
        setLoading(false)
      })
  }, [])

  const onAskSubmit = useConstCallback(async () => {
    if (authors.length == 0) {
      setError(t('articles.authorship.needs-one-author'))
      return
    }
    if (user && originAuthors.includes(user) && !authors.includes(user)) {
      setAskTransferOwnership(true)
    } else {
      await onSubmit()
    }
  })

  const onSubmit = useConstCallback(async () => {
    setSaving(true)
    setError('')
    setSavingSuccess(false)

    const input = {
      pageId: pageId,
      authorsIds: authors.map(x => x.id).filter(x => x !== undefined),
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

  const onCancelTransferOwnership = useConstCallback(() => {
    setAskTransferOwnership(false)
  })

  const onClear = useConstCallback(e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    setAuthors([])
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

  const onChange = useConstCallback((authors: UserData[]) => {
    setAuthors(authors)
  })

  return (
    <Styles>
      {saving && (
        <WikidotModal isLoading>
          <p>{t('articles.authorship.saving')}</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>{t('articles.authorship.saved')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('articles.authorship.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.authorship.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      {askTransferOwnership && (
        <WikidotModal
          buttons={[
            { title: t('articles.authorship.disown-cancel'), onClick: onCancelTransferOwnership },
            { title: t('articles.authorship.disown-confirm'), onClick: onSubmit },
          ]}
        >
          <h1>{t('articles.authorship.disown-title')}</h1>
          <p>
            {t('articles.authorship.disown-note')}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.authorship.close')}
      </a>
      <h1>{t('articles.authorship.title')}</h1>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.authorship.author-label')}</td>
            </tr>
            <tr>
              <td className="w-authorship-editor-container">
                {loading && <Loader className="loader" />}
                <AuthorshipEditorComponent authors={authors} allUsers={allUsers} onChange={onChange} editable={editable} />
              </td>
            </tr>
          </tbody>
        </table>
        {editable && (
          <div className="buttons form-actions">
            <input type="button" className="btn btn-danger" value={t('articles.authorship.close')} onClick={onCancel} />
            <input type="button" className="btn btn-default" value={t('articles.authorship.clear')} onClick={onClear} />
            <input type="button" className="btn btn-primary" value={t('articles.authorship.save')} onClick={onAskSubmit} />
          </div>
        )}
      </form>
    </Styles>
  )
}

export default ArticleAuthorship
