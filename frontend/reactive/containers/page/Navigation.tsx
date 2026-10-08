import { t } from '~util/i18n'
import React from 'react'
import { Paths } from '~reactive/paths'
import { useConfigContext } from '~reactive/config'
import * as Styled from './Navigation.styles'

const Navigation: React.FC = () => {
  const { user } = useConfigContext()
  return (
    <Styled.Container>
      <Styled.ExternalLink href={`/-/users/${user.urlName || user.username}`}>{t('page.navigation.profile')}</Styled.ExternalLink>
      <Styled.ExternalLink href="/-/profile/edit">{t('page.navigation.edit-profile')}</Styled.ExternalLink>
      <Styled.Link to={Paths.notifications}>{t('page.navigation.notifications')}</Styled.Link>
      <Styled.Link to={Paths.messages}>{t('page.navigation.messages')}</Styled.Link>
      <Styled.Link to={Paths.favourites}>{t('page.navigation.favourites')}</Styled.Link>
      <Styled.Link to={Paths.ratings}>{t('page.navigation.ratings')}</Styled.Link>
      <Styled.Link to={Paths.likedPosts}>{t('page.navigation.liked-posts')}</Styled.Link>
      <Styled.Link to={Paths.myTickets}>{t('page.navigation.my-tickets')}</Styled.Link>
    </Styled.Container>
  )
}

export default Navigation
