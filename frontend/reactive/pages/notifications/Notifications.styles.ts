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

export const FilterContainer = styled.div`
  display: flex;
  gap: 4px;
  margin-bottom: 12px;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  text-transform: none;
  letter-spacing: 0;
`

export const FilterLabel = styled.span`
  padding: 6px 4px 6px 0;
  color: ${({ theme }) => theme.uiForeground};
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
      color: ${theme.primaryForeground};
      border-color: ${theme.primary};
      background: ${theme.primary};
      font-weight: 600;
    `};

  &:hover {
    color: ${({ checked, theme }) => (checked ? theme.primaryForeground : theme.foreground)};
    border-color: ${({ checked, theme }) => (checked ? theme.primary : theme.quiet)};
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

export const Toolbar = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 8px 0 12px;
  flex-wrap: wrap;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  text-transform: none;
  letter-spacing: 0;
  color: ${({ theme }) => theme.uiForeground};
`

export const ToolbarAction = styled.button<{ danger?: boolean }>`
  font: inherit;
  letter-spacing: inherit;
  text-transform: inherit;
  padding: 6px 10px;
  cursor: pointer;
  background: transparent;
  border-radius: 2px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  color: ${({ danger, theme }) => (danger ? '#c92a2a' : theme.foreground)};

  &:disabled {
    opacity: 0.4;
    cursor: default;
  }

  &:not(:disabled):hover {
    background: ${({ theme }) => theme.windowPadding};
  }
`

export const ToolbarLabel = styled.label`
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
`

export const SelectRow = styled.div`
  display: flex;
  align-items: flex-start;
  gap: 10px;

  > input {
    margin-top: 20px;
  }

  > *:last-child {
    flex: 1;
    min-width: 0;
  }
`
