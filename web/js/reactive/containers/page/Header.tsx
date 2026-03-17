import React from 'react'
import * as Styled from './Header.styles'

interface Props {
  title: string
}

const Header: React.FC<Props> = ({ title }) => {
  return (
    <Styled.Container>
      <Styled.FixedWidthContainer>
        <Styled.Heading>{title}</Styled.Heading>
        <Styled.GoBack href="/">
          <span className="fa fa-arrow-left"></span> 返回网站首页
        </Styled.GoBack>
      </Styled.FixedWidthContainer>
      <Styled.Border />
    </Styled.Container>
  )
}

export default Header
