import styled from 'styled-components'

export const SectionHead = styled.div`
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
`

export const Kicker = styled.div`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
  letter-spacing: 0.01em;
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
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
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
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
`

export const Added = styled.span`
  margin-left: auto;
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
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
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
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
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 2px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  color: ${({ positive, theme }) => (positive ? theme.foreground : theme.uiForeground)};
`
