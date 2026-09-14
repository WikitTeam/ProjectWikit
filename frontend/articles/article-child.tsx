import { t } from '~util/i18n'
import * as React from 'react'
import { useRef, useState } from 'react'
import styled from 'styled-components'
import useConstCallback from '../util/const-callback'
import { isFullNameAllowed } from '../util/validate-article-name'
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

const ArticleChild: React.FC<Props> = ({ pageId, onClose }) => {
  const [child, setChild] = useState('')
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement | null>(null)

  const onSubmit = useConstCallback(async e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }

    if (isFullNameAllowed(child) && child != pageId) {
      window.location.href = `/${child}/edit/true/parent/${pageId}`
    } else {
      setError(t('articles.child.invalid-name'))
    }
  })

  const onCancel = useConstCallback(e => {
    if (e) {
      e.preventDefault()
      e.stopPropagation()
    }
    if (onClose) onClose()
  })

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    switch (e.target.name) {
      case 'child':
        setChild(e.target.value)
        break
    }
  }

  const onCloseError = () => {
    setError('')
  }

  const onSnippet = useConstCallback((e: React.MouseEvent, value: string) => {
    e.preventDefault()
    e.stopPropagation()
    inputRef.current?.focus()
    setChild(value)
  })

  return (
    <Styles>
      {error && (
        <WikidotModal buttons={[{ title: t('articles.child.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('articles.child.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        {t('articles.child.close')}
      </a>
      <h1>{t('articles.child.title')}</h1>
      <p>{t('articles.child.note')}</p>
      <p>
        {' '}
        <em>{t('articles.child.hint-label')}</em> <a onClick={e => onSnippet(e, 'fragment:')}>fragment:</a> /{' '}
        <a onClick={e => onSnippet(e, `fragment:${pageId}_`)}>{`fragment:${pageId}_`}</a>
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>{t('articles.child.parent-name-label')}</td>
              <td>{pageId}</td>
            </tr>
            <tr>
              <td>{t('articles.child.child-name-label')}</td>
              <td>
                <input ref={inputRef} type="text" name="child" className="text" onChange={onChange} id="page-child-input" value={child} autoFocus />
              </td>
            </tr>
          </tbody>
        </table>
        <div className="buttons form-actions">
          <input type="button" className="btn btn-danger" value={t('articles.child.cancel')} onClick={onCancel} />
          <input type="button" className="btn btn-primary" value={t('articles.child.create')} onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleChild
