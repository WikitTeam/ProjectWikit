import { Link } from 'react-router-dom'
import styled from 'styled-components'

export const SectionHead = styled.div`
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
`

export const Kicker = styled.div`
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
  letter-spacing: 0;
  line-height: 1.4;

  b {
    color: ${({ theme }) => theme.foreground};
    font-weight: 500;
  }

  .sep {
    color: ${({ theme }) => theme.windowStrong};
    margin: 0 6px;
  }
`

export const H1 = styled.h1`
  font-size: 32px;
  font-weight: 500;
  line-height: 1.1;
  letter-spacing: -0.02em;
  margin: 0;
  color: ${({ theme }) => theme.foreground};
`

export const Count = styled.div`
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  text-transform: none;
  letter-spacing: 0;
  color: ${({ theme }) => theme.uiForeground};
  margin-bottom: 12px;
`

export const List = styled.div`
  display: flex;
  flex-direction: column;
`

export const Item = styled.div`
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 12px 0;
  border-bottom: 1px solid ${({ theme }) => theme.windowPadding};
`

export const Title = styled.a`
  font-size: 15px;
  color: ${({ theme }) => theme.foreground};
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
`

export const Name = styled.span`
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
`

export const Added = styled.span`
  margin-left: auto;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
`

export const LoaderContainer = styled.div`
  display: flex;
  justify-content: center;
  padding: 24px 0;
`

export const EmptyMessage = styled.div`
  padding: 24px 0;
  color: ${({ theme }) => theme.uiForeground};
`

export const Pager = styled.div`
  display: flex;
  gap: 4px;
  margin-top: 16px;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
`

export const PagerStep = styled.button<{ current?: boolean }>`
  padding: 6px 10px;
  cursor: pointer;
  background: ${({ current, theme }) => (current ? theme.windowPadding : 'transparent')};
  color: ${({ current, theme }) => (current ? theme.foreground : theme.uiForeground)};
  border: 1px solid ${({ current, theme }) => (current ? theme.windowStrong : 'transparent')};
  border-radius: 2px;

  &:disabled {
    cursor: default;
  }

  &:not(:disabled):hover {
    color: ${({ theme }) => theme.foreground};
  }
`

export const PagerDots = styled.span`
  padding: 6px 2px;
  color: ${({ theme }) => theme.uiForeground};
`

export const Badge = styled.span<{ positive: boolean }>`
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  padding: 2px 6px;
  border-radius: 2px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  color: ${({ positive, theme }) => (positive ? theme.foreground : theme.uiForeground)};
`

export const Ticket = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 0;
  border-bottom: 1px solid ${({ theme }) => theme.windowPadding};
`

export const TicketHead = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 10px;
  font-size: 15px;
  color: ${({ theme }) => theme.foreground};
`

export const TicketKind = styled.span`
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
`

export const TicketReply = styled.div`
  white-space: pre-wrap;
  font-size: 14px;
  color: ${({ theme }) => theme.uiForeground};
`

export const TicketTitle = styled(Link)`
  color: ${({ theme }) => theme.accent};
  font-weight: 600;
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.accent};
    outline-offset: 2px;
  }
`
