import React from 'react'
import styled from 'styled-components'
import { renderTo, unmountFromRoot } from '~util/react-render-into'
import { UserData } from '../api/user'
import useConstCallback from './const-callback'
import formatDate from './date-format'
import UserView from './user-view'

interface Button {
  title: string
  onClick: () => void
}

interface Props {
  buttons?: Array<Button>
  background?: string
  children?: React.ReactNode
}

const Styles = styled.div`
  .w-message {
    position: absolute;
    right: 2em;
    border: 1px dashed #888;
    padding: 0.5em 1em;
    max-width: 20em;
    opacity: 0.9;
    z-index: 999999;
  }
`

const WikidotMessage: React.FC<Props> = ({ children, buttons, background }: Props) => {
  const handleCallback = useConstCallback((e, callback) => {
    e.preventDefault()
    e.stopPropagation()
    callback()
  })

  return (
    <Styles>
      <div className="w-message" style={{ background: background }}>
        {children}
        {buttons && <br />}
        {buttons?.map((button, i) => (
          <React.Fragment key={i}>
            <a onClick={e => handleCallback(e, button.onClick)}>{button.title}</a>
            {i !== buttons.length - 1 && ' | '}
          </React.Fragment>
        ))}
      </div>
    </Styles>
  )
}

function getMessageContainer() {
  return document.getElementById('action-area-top')!
}

function addMessage(message: React.ReactNode) {
  renderTo(getMessageContainer(), message)
}

export function removeMessage() {
  const node = getMessageContainer()
  unmountFromRoot(node)
}

export function showPreviewMessage() {
  const onDown = () => {
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    }, 1)
  }

  const onClose = () => {
    removeMessage()
  }

  const message = (
    <WikidotMessage
      background={'#FDD'}
      buttons={[
        { title: '向下到编辑器', onClick: onDown },
        { title: '关闭消息', onClick: onClose },
      ]}
    >
      提示：这只是预览。
      <br />
      如果现在关闭此页面，更改将不会保存。
    </WikidotMessage>
  )

  addMessage(message)
  setTimeout(() => {
    window.scrollTo(window.scrollX, document.body.scrollTop)
  }, 1)
}

export function showVersionMessage(num: number, date: Date, user: UserData, pageId: string) {
  const onDown = () => {
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    }, 1)
  }

  const onClose = () => {
    removeMessage()
  }

  const message = (
    <WikidotMessage
      background={'#EEF'}
      buttons={[
        { title: '向下到版本列表', onClick: onDown },
        { title: '关闭消息', onClick: onClose },
      ]}
    >
      <table>
        <tbody>
          <tr>
            <td>版本号：</td>
            <td>{num}</td>
          </tr>
          <tr>
            <td>创建日期：</td>
            <td>{formatDate(date)}</td>
          </tr>
          <tr>
            <td>作者：</td>
            <td>
              <UserView data={user} />
            </td>
          </tr>
          <tr>
            <td>页面名称：</td>
            <td>{pageId}</td>
          </tr>
        </tbody>
      </table>
    </WikidotMessage>
  )

  addMessage(message)
  setTimeout(() => {
    window.scrollTo(window.scrollX, document.body.scrollTop)
  }, 1)
}

export default WikidotMessage