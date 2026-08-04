import React from 'react'
import { Helmet } from 'react-helmet-async'
import Header from './Header'
import * as Styled from './Page.styles'

interface Props {
  title: string
  crumb?: string
  hasBorder?: boolean
  children?: React.ReactNode
}

export const Page: React.FC<Props> = ({ children, title, crumb }) => {
  return (
    <Styled.Container>
      <Helmet>
        <title>{title} - ProjectWikit</title>
      </Helmet>
      <Styled.RootStyles />
      <Header crumb={crumb || title} />
      {children}
    </Styled.Container>
  )
}

export default Page
