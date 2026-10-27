import React from 'react'
import { Helmet } from 'react-helmet-async'
import { useConfigContext } from '~reactive/config'
import Header from './Header'
import * as Styled from './Page.styles'

interface Props {
  title: string
  crumb?: string
  hasBorder?: boolean
  children?: React.ReactNode
}

export const Page: React.FC<Props> = ({ children, title, crumb }) => {
  const { site } = useConfigContext()
  const siteTitle = site?.title || 'ProjectWikit'
  return (
    <Styled.Container>
      <Helmet>
        <title>{`${title} - ${siteTitle}`}</title>
      </Helmet>
      <Styled.RootStyles />
      <Header crumb={crumb || title} />
      {children}
      <Styled.Footer>
        {site?.title && <Styled.FooterHome href="/">{site.title}</Styled.FooterHome>}
        <Styled.FooterCredit>
          Powered by{' '}
          <a href="https://github.com/WikitTeam/ProjectWikit/" target="_blank" rel="noopener">
            ProjectWikit
          </a>
        </Styled.FooterCredit>
      </Styled.Footer>
    </Styled.Container>
  )
}

export default Page
