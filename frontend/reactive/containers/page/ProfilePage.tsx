import { t } from '~util/i18n'
import React from 'react'
import Navigation from './Navigation'
import Page from './Page'
import * as Styled from './Page.styles'

interface Props {
  crumb?: string
  children?: React.ReactNode
}

export const ProfilePage: React.FC<Props> = ({ children, crumb }) => {
  return (
    <Page title={t('page.profile-page.title')} crumb={crumb || t('page.profile-page.title')}>
      <Navigation />
      <Styled.MainContainer>{children}</Styled.MainContainer>
    </Page>
  )
}

export default ProfilePage
