import { t } from '~util/i18n'
import React from 'react'
import { useConfigContext } from '~reactive/config'
import * as Styled from './Header.styles'

interface Props {
  crumb: string
}

const Header: React.FC<Props> = ({ crumb }) => {
  const { site } = useConfigContext()
  return (
    <Styled.Container>
      <Styled.Brand href="/">
        <Styled.BrandLogo src={site?.systemIcon || '/-/static/images/wikitHana.png'} alt="ProjectWikit" />
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
