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
      setError('无效的子页面ID!')
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
        <WikidotModal buttons={[{ title: '关闭', onClick: onCloseError }]} isError>
          <p>
            <strong>错误:</strong> {error}
          </p>
        </WikidotModal>
      )}
      <a className="action-area-close btn btn-danger" href="#" onClick={onCancel}>
        关闭
      </a>
      <h1>创建子页面</h1>
      <p>此操作将创建一个以此页面为父页面的新页面</p>
      <p>
        {' '}
        <em>提示:</em> <a onClick={e => onSnippet(e, 'fragment:')}>fragment:</a> /{' '}
        <a onClick={e => onSnippet(e, `fragment:${pageId}_`)}>{`fragment:${pageId}_`}</a>
      </p>

      <form method="POST" onSubmit={onSubmit}>
        <table className="form">
          <tbody>
            <tr>
              <td>此页面名称:</td>
              <td>{pageId}</td>
            </tr>
            <tr>
              <td>子页面名称:</td>
              <td>
                <input ref={inputRef} type="text" name="child" className="text" onChange={onChange} id="page-child-input" value={child} autoFocus />
              </td>
            </tr>
          </tbody>
        </table>
        <div className="buttons form-actions">
          <input type="button" className="btn btn-danger" value="关闭" onClick={onCancel} />
          <input type="button" className="btn btn-primary" value="创建" onClick={onSubmit} />
        </div>
      </form>
    </Styles>
  )
}

export default ArticleChild
