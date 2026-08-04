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
    <Page title="用户界面" crumb={crumb || '用户界面'}>
      <Navigation />
      <Styled.MainContainer>{children}</Styled.MainContainer>
    </Page>
  )
}

export default ProfilePage
