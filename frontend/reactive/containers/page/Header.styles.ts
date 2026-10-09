import styled from 'styled-components'

export const Container = styled.div`
  padding: 14px 24px;
  display: flex;
  align-items: center;
  gap: 14px;
  background: ${({ theme }) => theme.bar};
`

export const Brand = styled.a`
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: ${({ theme }) => theme.barForeground};

  &:hover { text-decoration: none; }
`

export const BrandLogo = styled.img`
  width: 22px;
  height: 22px;
`

export const Wordmark = styled.span`
  font-weight: 600;
  letter-spacing: -0.01em;
  font-size: 15px;
`

export const Divider = styled.span`
  width: 1px;
  height: 18px;
  background: rgba(255, 255, 255, 0.28);
`

export const Path = styled.span`
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: ${({ theme }) => theme.barMuted};

  b {
    color: ${({ theme }) => theme.barForeground};
    font-weight: 600;
  }

  .sep {
    color: #7c8a9c;
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
  color: #dce3eb;
  text-decoration: none;

  &:hover {
    color: ${({ theme }) => theme.barForeground};
    text-decoration: underline;
    text-underline-offset: 3px;
  }
`
