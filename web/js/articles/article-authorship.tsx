import * as React from 'react'
import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { fetchArticle, updateArticle } from '../api/articles'
import { fetchAllUsers, UserData } from '../api/user'
import AuthorshipEditorComponent from '../components/authorship-editor'
import sleep from '../util/async-sleep'
import useConstCallback from '../util/const-callback'
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
        setError(e.error || '连接服务器失败')
      })
      .finally(() => {
        setLoading(false)
      })
  }, [])

  const onAskSubmit = useConstCallback(async () => {
    if (authors.length == 0) {
      setError('必须至少指定一个作者')
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
      setError(e.error || '连接服务器失败')
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
          <p>保存中...</p>
        </WikidotModal>
      )}
      {savingSuccess && (
        <WikidotModal>
          <p>保存成功！</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: '关闭', onClick: onCloseError }]} isError>
          <p>
            <strong>错误：</strong> {error}
          </p>
        </WikidotModal>
      )}
      {askTransferOwnership && (
        <WikidotModal
          buttons={[
            { title: '取消', onClick: onCancelTransferOwnership },
            { title: '是，我要放弃', onClick: onSubmit },
          ]}
        >
          <h1>是否放弃页面作者身份?</h1>
          <p>
            请注意，只有该页面作者栏中指定的人或管理员才能将作者权归还给您
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        关闭
      </a>
      <h1>页面的作者权</h1>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>作者:</td>
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
            <input type="button" className="btn btn-danger" value="关闭" onClick={onCancel} />
            <input type="button" className="btn btn-default" value="清除" onClick={onClear} />
            <input type="button" className="btn btn-primary" value="保存" onClick={onAskSubmit} />
          </div>
        )}
      </form>
    </Styles>
  )
}

export default ArticleAuthorship
