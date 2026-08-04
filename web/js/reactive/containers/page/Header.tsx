import React from 'react'
import * as Styled from './Header.styles'

interface Props {
  crumb: string
}

const Header: React.FC<Props> = ({ crumb }) => {
  return (
    <Styled.Container>
      <Styled.Brand href="/">
        <Styled.BrandLogo src="/-/static/images/wikitHana.png" alt="ProjectWikit" />
        <Styled.Wordmark>ProjectWikit</Styled.Wordmark>
      </Styled.Brand>
      <Styled.Divider />
      <Styled.Path>
        <b>{crumb}</b>
      </Styled.Path>
      <Styled.Spacer />
      <Styled.GoHome href="/">返回站点</Styled.GoHome>
    </Styled.Container>
  )
}

export default Header
