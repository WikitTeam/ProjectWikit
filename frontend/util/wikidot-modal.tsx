import Trans from '~util/trans'
import { t } from '~util/i18n'
import React, { useEffect, useState } from 'react'
import * as ReactDOM from 'react-dom'
import styled from 'styled-components'
import { v4 as uuid4 } from 'uuid'
import { renderTo, unmountFromRoot } from '~util/react-render-into'
import { ArticleLogEntry, revertArticleRevision } from '../api/articles'
import useConstCallback from './const-callback'

interface Button {
  title: string
  onClick: () => void
  type?: 'primary' | 'danger' | 'default'
}

interface Props {
  isLoading?: boolean
  isError?: boolean
  buttons?: Array<Button>
  children?: React.ReactNode
}

const WikidotModal: React.FC<Props> = ({ children, isLoading, isError, buttons }) => {
  const [modalId, setModalId] = useState('')

  useEffect(() => {
    const newModalId = uuid4()
    setModalId(newModalId)
    addModalContainer(newModalId)

    return () => {
      removeModalContainer(newModalId)
    }
  }, [])

  // 返回portal时如果没有显式的"any"类型，会导致与旧版本的兼容性问题

  const container = getModalContainer(modalId)
  if (container)
    return ReactDOM.createPortal(
      <WikidotModalWindow children={children} isLoading={isLoading} isError={isError} buttons={buttons} id={modalId} />,
      container,
    )
  return null
}

//
const Styles = styled.div`
  .odialog-container {
    position: fixed;
    z-index: 9999;
    left: 0;
    top: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #0000007f;
  }
  .owindow {
    max-width: 50em;
    width: auto;
  }
  .buttons-hr {
    margin-left: 0;
    margin-right: 0;
  }
  .w-modal-buttons {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
  }
  .owindow.owait .content {
    background-image: url(/-/static/images/progressbar.gif);
  }
`

const WikidotModalWindow: React.FC<Props & { id: string }> = ({ children, isLoading, isError, buttons, id }) => {
  const handleCallback = useConstCallback((e: React.MouseEvent, callback: () => void) => {
    e.preventDefault()
    e.stopPropagation()
    callback()
  })

  return (
    <Styles>
      <div className="odialog-container">
        <div id={`owindow-${id}`} className={`owindow ${isLoading ? 'owait' : ''} ${isError ? 'error' : ''}`}>
          <div className="content modal-body">
            {children}
            {buttons && (
              <>
                <hr className="buttons-hr" />
                <div className="w-modal-buttons button-bar modal-footer">
                  {buttons.map((button, i) => (
                    <a
                      key={i}
                      href="javascript:;"
                      className={`btn btn-${button.type || 'default'} button button-close-message`}
                      onClick={e => handleCallback(e, button.onClick)}
                    >
                      {button.title}
                    </a>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </Styles>
  )
}

function getModalContainerElement() {
  return document.getElementById('w-modals')
}

function addModalContainer(id: string) {
  const node = document.createElement('div')
  node.setAttribute('id', `modal-${id}`)
  getModalContainerElement()?.appendChild(node)
  return node
}

function getModalContainer(id: string) {
  return getModalContainerElement()?.querySelector(`#modal-${id}`)
}

function removeModalContainer(id: string) {
  const node = getModalContainer(id)
  if (node && node.parentNode) {
    node?.parentNode.removeChild(node)
  }
}

export function addUnmanagedModal(modal: React.ReactNode) {
  const id = uuid4()
  const container = addModalContainer(id)
  renderTo(container, modal)
  return id
}

export function updateUnmanagedModal(id: string, modal: React.ReactNode) {
  const container = getModalContainer(id)
  if (container) {
    renderTo(container, modal)
  }
}

export function removeUnmanagedModal(id: string) {
  const container = getModalContainer(id)
  if (container) {
    unmountFromRoot(container)
  }
  removeModalContainer(id)
}

export function showErrorModal(error: string) {
  let uuid: string | null = null

  const onCloseError = () => {
    if (!uuid) {
      return
    }
    removeUnmanagedModal(uuid)
  }

  const modal = (
    <WikidotModal buttons={[{ title: t('util.wikidot-modal.error-dismiss'), onClick: onCloseError }]} isError>
      <p>
        <strong>{t('util.wikidot-modal.error-label')}</strong> {error}
      </p>
    </WikidotModal>
  )

  uuid = addUnmanagedModal(modal)
}

export function showRevertModal(pageId: string, entry: ArticleLogEntry) {
  let uuid: string | null = null

  const onClose = () => {
    if (!uuid) {
      return
    }
    removeUnmanagedModal(uuid)
  }

  const onRevert = () => {
    onClose()
    revertArticleRevision(pageId, entry.revNumber)
      .then(function (pageData) {
        window.location.href = `/${pageData.pageId}`
      })
      .catch(error => {
        showErrorModal(error.toString())
      })
  }

  const modal = (
    <WikidotModal
      buttons={[
        { title: t('util.wikidot-modal.revert-cancel'), onClick: onClose },
        { title: t('util.wikidot-modal.revert-confirm-button'), onClick: onRevert },
      ]}
    >
      <h1>{t('util.wikidot-modal.revert-title')}</h1>
      <p>
        <Trans id="util.wikidot-modal.revert-confirm" children={{ revision: <strong>#{entry.revNumber}</strong> }} />
      </p>
    </WikidotModal>
  )

  uuid = addUnmanagedModal(modal)
}

export default WikidotModal