import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getNotification, Notification as INotification } from '~api/notifications'
import { ProfilePage } from '~reactive/containers/page'
import { Paths } from '~reactive/paths'
import { t } from '~util/i18n'
import Notification from './Notification'
import * as Styled from './Notifications.styles'

const NotificationView: React.FC<{ id: number }> = ({ id }) => {
  const [found, setFound] = useState<INotification | null>()
  useEffect(() => {
    getNotification(id)
      .then(resp => setFound(resp.notifications[0] ?? null))
      .catch(() => setFound(null))
  }, [id])

  return (
    <ProfilePage crumb={t('notifications.crumb')}>
      <Styled.SectionHead>
        <Link to={Paths.notifications}>{t('notifications.back')}</Link>
      </Styled.SectionHead>
      {found && <Notification notification={found} full />}
      {found === null && <p>{t('notifications.not-found')}</p>}
    </ProfilePage>
  )
}

export default NotificationView
