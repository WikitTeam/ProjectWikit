import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { Paths } from '~reactive/paths'
import { t } from '~util/i18n'
import { UnreadCounts } from '../api/notifications'
import { UserData } from '../api/user'
import useConstCallback from '../util/const-callback'
import { DEFAULT_AVATAR } from '../util/user-view'
import { showErrorModal } from '../util/wikidot-modal'

interface Props {
  user: UserData
  notificationCount: number
  unread?: UnreadCounts
  signupClosed?: boolean
}

const PageLoginStatus: React.FC<Props> = ({ user, notificationCount, unread, signupClosed }: Props) => {
  const withCount = (label: string, count?: number) => (count ? t('page-login-status.with-count', { label, count }) : label)
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

  const onSignUp = useConstCallback(e => {
    if (!signupClosed) {
      return
    }
    e.preventDefault()
    showErrorModal(t('page-login-status.signup-closed'))
  })

  if (user.type === 'anonymous') {
    return (
      <>
        <a className="login-status-create-account btn" href="/-/signup" onClick={onSignUp}>
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
          <a href={`/-/users/${user.urlName || user.username}`}>
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
                <a href={`/-/notifications`}>{withCount(t('page-login-status.notifications'), unread?.notices)}</a>
              </li>
              <li>
                <a href={`/-/notifications/comments`}>{withCount(t('page-login-status.comments'), unread?.comments)}</a>
              </li>
              <li>
                <a href={`/-/messages`}>{withCount(t('page-login-status.messages'), unread?.messages)}</a>
              </li>
              <li>
                <a href={`/-/favourites`}>{t('page-login-status.favourites')}</a>
              </li>
              <li>
                <a href={`/-/ratings`}>{t('page-login-status.ratings')}</a>
              </li>
              <li>
                <a href={`/-/liked-posts`}>{t('page-login-status.liked-posts')}</a>
              </li>
              <li>
                <a href={`/-/my-tickets`}>{t('page-login-status.my-tickets')}</a>
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
