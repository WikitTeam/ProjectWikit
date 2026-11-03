import React from 'react'
import { matchPath, useLocation } from 'react-router-dom'
import { useConfigContext } from '~reactive/config'
import { Paths } from '~reactive/paths'
import { t } from '~util/i18n'
import * as Styled from './Navigation.styles'

const withCount = (label: string, count?: number) => (count ? t('page.navigation.with-count', { label, count }) : label)

const Navigation: React.FC = () => {
  const { user, unread } = useConfigContext()
  const { pathname } = useLocation()
  const onComments = Boolean(matchPath(`${Paths.comments}/*`, pathname))
  const onNotices = !onComments && Boolean(matchPath(`${Paths.notifications}/*`, pathname))
  return (
    <Styled.Container>
      <Styled.ExternalLink href={`/-/users/${user.urlName || user.username}`}>{t('page.navigation.profile')}</Styled.ExternalLink>
      <Styled.ExternalLink href="/-/profile/edit">{t('page.navigation.edit-profile')}</Styled.ExternalLink>
      <Styled.Link to={Paths.notifications} end className={onNotices ? 'active' : undefined}>
        {withCount(t('page.navigation.notifications'), unread?.notices)}
      </Styled.Link>
      <Styled.Link to={Paths.comments}>{withCount(t('page.navigation.comments'), unread?.comments)}</Styled.Link>
      <Styled.Link to={Paths.messages}>{withCount(t('page.navigation.messages'), unread?.messages)}</Styled.Link>
      <Styled.Link to={Paths.favourites}>{t('page.navigation.favourites')}</Styled.Link>
      <Styled.Link to={Paths.ratings}>{t('page.navigation.ratings')}</Styled.Link>
      <Styled.Link to={Paths.likedPosts}>{t('page.navigation.liked-posts')}</Styled.Link>
      <Styled.Link to={Paths.myTickets}>{t('page.navigation.my-tickets')}</Styled.Link>
    </Styled.Container>
  )
}

export default Navigation
