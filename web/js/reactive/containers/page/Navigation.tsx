import { t } from '~util/i18n'
import React from 'react'
import { Paths } from '~reactive/paths'
import * as Styled from './Navigation.styles'

const Navigation: React.FC = () => {
  return (
    <Styled.Container>
      <Styled.Link to={Paths.profile}>{t('page.navigation.edit-profile')}</Styled.Link>
      <Styled.Link to={Paths.notifications}>{t('page.navigation.notifications')}</Styled.Link>
      <Styled.Link to={Paths.messages}>{t('page.navigation.messages')}</Styled.Link>
    </Styled.Container>
  )
}

export default Navigation
