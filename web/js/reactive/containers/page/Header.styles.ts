import styled from 'styled-components'

export const Container = styled.div`
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  padding: 14px 24px;
  display: flex;
  align-items: center;
  gap: 14px;
  background: ${({ theme }) => theme.windowPadding};
`

export const Brand = styled.a`
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: ${({ theme }) => theme.foreground};

  &:hover { text-decoration: none; }
`

export const BrandLogo = styled.img`
  width: 22px;
  height: 22px;
`

export const Wordmark = styled.span`
  font-weight: 500;
  letter-spacing: -0.01em;
  font-size: 15px;
`

export const Divider = styled.span`
  width: 1px;
  height: 18px;
  background: ${({ theme }) => theme.windowStrong};
`

export const Path = styled.span`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 12px;
  color: ${({ theme }) => theme.uiForeground};

  b {
    color: ${({ theme }) => theme.foreground};
    font-weight: 500;
  }

  .sep {
    color: ${({ theme }) => theme.windowStrong};
    margin: 0 6px;
  }

  @media (max-width: 720px) {
    display: none;
  }
`

export const Spacer = styled.div`
  flex: 1;
`

export const GoHome = styled.a`
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
  text-decoration: none;

  &:hover {
    color: ${({ theme }) => theme.foreground};
    text-decoration: underline;
    text-underline-offset: 3px;
  }
`
