import { t } from '~util/i18n'
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
        <a className="login-status-create-account btn" href="/-/signup">
          {t('page-login-status.sign-up')}
        </a>{' '}
        <span>{t('page-login-status.or')}</span>{' '}
        <a className="login-status-sign-in btn btn-primary" href={`/-/login?to=${encodeURIComponent(window.location.href)}`}>
          {t('page-login-status.sign-in')}
        </a>
      </>
    )
  } else {
    return (
      <>
        <span className="printuser w-user">
          <a href={`/-/profile`}>
            <img className="small" src={user.avatar || DEFAULT_AVATAR} alt={user.name || user.username} />
          </a>
          {user.name || user.username}
        </span>
        {(user.admin || user.staff) && (
          <>
            {' '}|{' '}
            <a id="w-admin-link" href={`/-/admin`} target="_blank">
              {t('page-login-status.admin')}
            </a>
          </>
        )}
        {' | '}
        <a id="my-account" href={`/-/users/${user.urlName || user.username}`}>
          {t('page-login-status.profile')}
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
                <a href={`/-/notifications`}>{t('page-login-status.notifications')}</a>
              </li>
              <li>
                <a href={`/-/messages`}>{t('page-login-status.messages')}</a>
              </li>
              <li>
                <a href={`/-/profile/edit`}>{t('page-login-status.settings')}</a>
              </li>
              <li>
                <a href={`/-/logout?to=${encodeURIComponent(window.location.href)}`}>{t('page-login-status.sign-out')}</a>
              </li>
            </ul>
          </div>
        )}
      </>
    )
  }
}

export default PageLoginStatus
