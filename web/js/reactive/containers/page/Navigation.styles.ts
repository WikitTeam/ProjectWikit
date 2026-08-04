import { NavLink } from 'react-router-dom'
import styled from 'styled-components'

export const Container = styled.div`
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  padding: 0 24px;
  display: flex;
  gap: 4px;
  align-items: flex-end;
  min-height: 44px;
  background: ${({ theme }) => theme.windowPadding};
  overflow-x: auto;
`

export const Link = styled(NavLink)`
  padding: 12px 14px;
  text-decoration: none;
  color: ${({ theme }) => theme.uiForeground};
  font-size: 14px;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;

  &:link, &:hover, &:active, &:visited { text-decoration: none; }

  &:hover {
    color: ${({ theme }) => theme.foreground};
  }

  &.active, &.active:hover {
    color: ${({ theme }) => theme.foreground};
    border-bottom-color: ${({ theme }) => theme.foreground};
    font-weight: 500;
  }
`
