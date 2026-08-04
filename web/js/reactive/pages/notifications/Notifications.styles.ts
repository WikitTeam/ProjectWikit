import styled, { css } from 'styled-components'

export const Container = styled.div`
  margin: 0;
  padding: 0;
`

export const List = styled.div`
  display: flex;
  flex-direction: column;
`

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

export const FilterContainer = styled.div`
  display: flex;
  gap: 4px;
  margin-bottom: 12px;
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
`

export const RadioLabel = styled.label<{ checked: boolean }>`
  padding: 6px 10px;
  color: ${({ theme }) => theme.uiForeground};
  cursor: pointer;
  border: 1px solid transparent;
  border-radius: 2px;

  ${({ checked, theme }) =>
    checked &&
    css`
      color: ${theme.foreground};
      border-color: ${theme.windowStrong};
      background: ${theme.windowPadding};
    `};

  &:hover {
    color: ${({ theme }) => theme.foreground};
  }
`

export const RadioInput = styled.input`
  display: none;
`

export const EmptyMessage = styled.div`
  text-align: center;
  padding: 40px 16px;
  color: ${({ theme }) => theme.uiForeground};
  font-size: 15px;
`

export const LoaderContainer = styled.div`
  padding: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: ${({ theme }) => theme.uiForeground};
  font-size: 13px;
`
