import * as React from 'react'
import { useState } from 'react'
import { matchPath, useLocation, useNavigate } from 'react-router-dom'
import { NotificationKind } from '~api/notifications'
import { useConfigContext } from '~reactive/config'
import { ProfilePage } from '~reactive/containers/page'
import NotificationsInfiniteScroll from '~reactive/pages/notifications/NotificationsInfiniteScroll'
import { Paths } from '~reactive/paths'
import { t } from '~util/i18n'
import useConstCallback from '../../../util/const-callback'
import * as Styled from './Notifications.styles'
import NotificationView from './NotificationView'

const NOTICE_KINDS: NotificationKind[] = ['notices', 'new_ticket']
const COMMENT_KINDS: NotificationKind[] = ['comments', 'replies', 'forum_mention', 'post_like']

const kindLabel = (kind: NotificationKind) => t(`notifications.kind-${kind === 'notices' || kind === 'comments' ? 'all' : kind}`)

const Notifications: React.FC = () => {
  const { pathname } = useLocation()
  const viewing = matchPath(`${Paths.notifications}/view/:id`, pathname)
  if (viewing) return <NotificationView key={viewing.params.id} id={Number(viewing.params.id)} />
  const onComments = Boolean(matchPath(`${Paths.comments}/*`, pathname))
  return <Section key={onComments ? 'comments' : 'notices'} onComments={onComments} pathname={pathname} />
}

const Section: React.FC<{ onComments: boolean; pathname: string }> = ({ onComments, pathname }) => {
  const [forceUpdate, setForceUpdate] = useState<boolean>(false)
  const { reviewsTickets } = useConfigContext()
  const kinds = onComments ? COMMENT_KINDS : NOTICE_KINDS.filter(one => reviewsTickets || one !== 'new_ticket')
  const [kind, setKind] = useState<NotificationKind>(kinds[0])
  const showUnread = Boolean(
    matchPath(onComments ? Paths.comments : Paths.notifications, pathname) ||
      matchPath(onComments ? Paths.commentsUnread : Paths.notificationsUnread, pathname),
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
    <ProfilePage crumb={onComments ? t('notifications.comments-title') : t('notifications.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('notifications.breadcrumb-profile')}</b>
          <span className="sep">/</span>
          {onComments ? t('notifications.comments-title') : t('notifications.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{onComments ? t('notifications.comments-title') : t('notifications.title')}</Styled.H1>
      </Styled.SectionHead>
      <Styled.FilterContainer>
        <Styled.RadioLabel checked={showUnread}>
          <Styled.RadioInput
            type="radio"
            name="filter"
            checked={showUnread}
            onChange={() => onChecked(onComments ? Paths.commentsUnread : Paths.notificationsUnread)}
          />
          {t('notifications.tab-unread')}
        </Styled.RadioLabel>
        <Styled.RadioLabel checked={!showUnread}>
          <Styled.RadioInput
            type="radio"
            name="filter"
            checked={!showUnread}
            onChange={() => onChecked(onComments ? Paths.commentsAll : Paths.notificationsAll)}
          />
          {t('notifications.tab-all')}
        </Styled.RadioLabel>
      </Styled.FilterContainer>
      {kinds.length > 1 && (
        <Styled.FilterContainer>
          <Styled.FilterLabel>{t('notifications.kind-label')}</Styled.FilterLabel>
          {kinds.map(one => (
            <Styled.RadioLabel checked={kind === one} key={one}>
              <Styled.RadioInput type="radio" name="kind" checked={kind === one} onChange={() => onKind(one)} />
              {kindLabel(one)}
            </Styled.RadioLabel>
          ))}
        </Styled.FilterContainer>
      )}
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
