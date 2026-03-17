import React from 'react'
import { Paths } from '~reactive/paths'
import * as Styled from './Navigation.styles'

const Navigation: React.FC = () => {
  return (
    <Styled.Container>
      <Styled.Link to={Paths.profile}>编辑个人资料</Styled.Link>
      <Styled.Link to={Paths.notifications}>通知</Styled.Link>
    </Styled.Container>
  )
}

export default Navigation
