import { t } from '~util/i18n'
import * as React from 'react'
import { useState } from 'react'
import { matchPath, useNavigate } from 'react-router-dom'
import { ProfilePage } from '~reactive/containers/page'
import { NotificationKind } from '~api/notifications'
import NotificationsInfiniteScroll from '~reactive/pages/notifications/NotificationsInfiniteScroll'
import { Paths } from '~reactive/paths'
import useConstCallback from '../../../util/const-callback'
import * as Styled from './Notifications.styles'

const KINDS: NotificationKind[] = ['all', 'post_like', 'replies', 'direct_message']

const Notifications: React.FC = () => {
  const [forceUpdate, setForceUpdate] = useState<boolean>(false)
  const [kind, setKind] = useState<NotificationKind>('all')
  const showUnread = Boolean(
    matchPath(`/-${Paths.notifications}`, window.location.pathname) || matchPath(`/-${Paths.notificationsUnread}`, window.location.pathname),
  )
  const navigate = useNavigate()

  const isForceUpdate = useConstCallback(() => {
    setForceUpdate(false)
    return forceUpdate
  })

  const onChecked = useConstCallback(dest => {
    navigate(dest)
    setForceUpdate(true)
  })

  const onKind = useConstCallback((next: NotificationKind) => {
    setKind(next)
    setForceUpdate(true)
  })

  return (
    <ProfilePage crumb={t('notifications.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('notifications.breadcrumb-profile')}</b><span className="sep">/</span>{t('notifications.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{t('notifications.title')}</Styled.H1>
      </Styled.SectionHead>
      <Styled.FilterContainer>
        <Styled.RadioLabel checked={showUnread}>
          <Styled.RadioInput type="radio" name="filter" checked={showUnread} onChange={() => onChecked(Paths.notificationsUnread)} />
          {t('notifications.tab-unread')}
        </Styled.RadioLabel>
        <Styled.RadioLabel checked={!showUnread}>
          <Styled.RadioInput type="radio" name="filter" checked={!showUnread} onChange={() => onChecked(Paths.notificationsAll)} />
          {t('notifications.tab-all')}
        </Styled.RadioLabel>
      </Styled.FilterContainer>
      <Styled.FilterContainer>
        {KINDS.map(one => (
          <Styled.RadioLabel checked={kind === one} key={one}>
            <Styled.RadioInput type="radio" name="kind" checked={kind === one} onChange={() => onKind(one)} />
            {t(`notifications.kind-${one}`)}
          </Styled.RadioLabel>
        ))}
      </Styled.FilterContainer>
      <NotificationsInfiniteScroll
        key={`tab-${showUnread}-${kind}`}
        batchSize={10}
        showUnread={showUnread}
        kind={kind}
        isForceUpdate={isForceUpdate}
      />
    </ProfilePage>
  )
}

export default Notifications
