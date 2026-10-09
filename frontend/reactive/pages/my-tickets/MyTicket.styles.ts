import { Link } from 'react-router-dom'
import styled from 'styled-components'

export const Back = styled(Link)`
  color: ${({ theme }) => theme.accent};
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
`

export const Facts = styled.dl`
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 8px 20px;
  margin: 0 0 24px;
  padding: 16px 0;
  border-top: 2px solid ${({ theme }) => theme.bar};
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  font-size: 14px;

  dt {
    font-weight: 600;
    color: ${({ theme }) => theme.uiForeground};
  }

  dd {
    margin: 0;
    color: ${({ theme }) => theme.foreground};
    font-variant-numeric: tabular-nums;
  }

  a {
    color: ${({ theme }) => theme.accent};
  }

  @media (max-width: 480px) {
    grid-template-columns: 1fr;
    gap: 2px;

    dd {
      margin-bottom: 8px;
    }
  }
`

export const Section = styled.section`
  margin-bottom: 24px;
`

export const Heading = styled.h2`
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 8px;
  color: ${({ theme }) => theme.foreground};
`

export const Text = styled.div`
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 15px;
  line-height: 1.6;
  color: ${({ theme }) => theme.foreground};
`

export const Reply = styled(Text)`
  padding: 12px 14px;
  border-left: 3px solid ${({ theme }) => theme.accent};
  background: ${({ theme }) => theme.uiBackground};
`

export const Messages = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0;

  li {
    padding: 10px 0;
    border-bottom: 1px solid ${({ theme }) => theme.windowPadding};
  }
`

export const MessageMeta = styled.div`
  display: flex;
  gap: 10px;
  font-size: 13px;
  font-weight: 600;
  color: ${({ theme }) => theme.foreground};
  margin-bottom: 4px;

  span {
    font-weight: 400;
    color: ${({ theme }) => theme.uiForeground};
    font-variant-numeric: tabular-nums;
  }
`
