import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { Paths } from '~reactive/paths'
import { UserData } from '../api/user'
import useConstCallback from '../util/const-callback'
import { DEFAULT_AVATAR } from '../util/user-view'

interface Props {
  user: UserData
  notificationCount: number
}

const PageLoginStatus: React.FC<Props> = ({ user, notificationCount }: Props) => {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    window.addEventListener('click', onPageClick)

    return () => {
      window.removeEventListener('click', onPageClick)
    }
  }, [])

  const onPageClick = useConstCallback(e => {
    let p = e.target
    while (p) {
      if (p === menuRef) {
        return
      }
      p = p.parentNode
    }
    setIsOpen(false)
  })

  const toggleMenu = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()

    setIsOpen(!isOpen)
  })

  if (user.type === 'anonymous') {
    return (
      <>
        <a className="login-status-create-account btn" href="/system:join">
          创建账户
        </a>{' '}
        <span>或</span>{' '}
        <a className="login-status-sign-in btn btn-primary" href={`/-/login?to=${encodeURIComponent(window.location.href)}`}>
          登录
        </a>
      </>
    )
  } else {
    return (
      <>
        <span className="printuser w-user">
          <a href={`/-/profile`}>
            <img className="small" src={user.avatar || DEFAULT_AVATAR} alt={user.username} />
          </a>
          {user.username}
        </span>
        {(user.admin || user.staff) && (
          <>
            {'\u00a0'}|{'\u00a0'}
            <a id="w-admin-link" href={`/-/admin`} target="_blank">
              管理面板
            </a>
          </>
        )}
        {'\u00a0|\u00a0'}
        <a id="my-account" href={`/-/users/${user.id}-${user.username}`}>
          个人资料
        </a>
        {notificationCount > 0 && (
          <>
            {' '}
            <a href={`/-${Paths.notificationsUnread}`}>
              <strong>({notificationCount})</strong>
            </a>
          </>
        )}
        <a id="account-topbutton" href="#" onClick={toggleMenu}>
          ▼
        </a>
        {isOpen && (
          <div id="account-options" ref={menuRef} style={{ display: 'block' }}>
            <ul>
              <li>
                <a href={`/-/notifications`}>通知</a>
              </li>
              <li>
                <a href={`/-/profile/edit`}>设置</a>
              </li>
              <li>
                <a href={`/-/logout?to=${encodeURIComponent(window.location.href)}`}>登出</a>
              </li>
            </ul>
          </div>
        )}
      </>
    )
  }
}

export default PageLoginStatus
