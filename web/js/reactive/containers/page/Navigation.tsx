import { t } from '~util/i18n'
import React from 'react'
import { Paths } from '~reactive/paths'
import * as Styled from './Navigation.styles'

const Navigation: React.FC = () => {
  return (
    <Styled.Container>
      <Styled.ExternalLink href="/-/profile/edit">{t('page.navigation.edit-profile')}</Styled.ExternalLink>
      <Styled.Link to={Paths.notifications}>{t('page.navigation.notifications')}</Styled.Link>
      <Styled.Link to={Paths.messages}>{t('page.navigation.messages')}</Styled.Link>
      <Styled.Link to={Paths.favourites}>{t('page.navigation.favourites')}</Styled.Link>
      <Styled.Link to={Paths.ratings}>{t('page.navigation.ratings')}</Styled.Link>
      <Styled.Link to={Paths.likedPosts}>{t('page.navigation.liked-posts')}</Styled.Link>
    </Styled.Container>
  )
}

export default Navigation
