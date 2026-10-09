import { NavLink } from 'react-router-dom'
import styled, { css } from 'styled-components'

export const Container = styled.div`
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  padding: 0 24px;
  display: flex;
  gap: 4px;
  align-items: flex-end;
  min-height: 44px;
  background: ${({ theme }) => theme.windowPadding};
  overflow-x: auto;
  overflow-y: hidden;
`

const linkStyle = css`
  padding: 12px 14px;
  text-decoration: none;
  color: ${({ theme }) => theme.uiForeground};
  font-size: 14px;
  font-weight: 500;
  border-bottom: 3px solid transparent;
  margin-bottom: -1px;
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;

  &:link, &:hover, &:active, &:visited { text-decoration: none; }

  &:hover {
    color: ${({ theme }) => theme.foreground};
    background: ${({ theme }) => theme.higlightBackground};
  }

  &.active, &.active:hover {
    color: ${({ theme }) => theme.primary};
    border-bottom-color: ${({ theme }) => theme.accent};
    font-weight: 600;
    background: transparent;
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.accent};
    outline-offset: -2px;
  }
`

export const Link = styled(NavLink)`${linkStyle}`

export const ExternalLink = styled.a`${linkStyle}`
