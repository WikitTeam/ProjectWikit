import { t } from '~util/i18n'
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
      <Styled.GoHome href="/">{t('page.header.back-to-site')}</Styled.GoHome>
    </Styled.Container>
  )
}

export default Header
