import styled from 'styled-components'

export const Folders = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
`

export const Folder = styled.button<{ current?: boolean }>`
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  padding: 6px 10px;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  border-radius: 2px;
  background: ${({ current, theme }) => (current ? theme.windowPadding : 'transparent')};
  color: ${({ current, theme }) => (current ? theme.foreground : theme.uiForeground)};
  border: 1px solid ${({ current, theme }) => (current ? theme.windowStrong : theme.windowPadding)};

  &:hover {
    color: ${({ theme }) => theme.foreground};
  }

  .count {
    font-variant-numeric: tabular-nums;
    color: ${({ theme }) => theme.uiForeground};
  }
`

export const Visibility = styled.span`
  font-size: 12px;
  color: ${({ theme }) => theme.uiForeground};
`

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
  margin-bottom: 4px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  font-size: 13px;
`

export const Action = styled.button<{ danger?: boolean }>`
  font: inherit;
  padding: 5px 10px;
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

export const NameInput = styled.input`
  min-width: 0;
  width: 180px;
  padding: 5px 8px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 2px;
  font: inherit;

  &:focus {
    outline: none;
    border-color: ${({ theme }) => theme.foreground};
  }
`

export const Check = styled.label`
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
`

export const Spacer = styled.span`
  flex: 1;
`

export const Move = styled.select`
  font: inherit;
  font-size: 12px;
  padding: 2px 4px;
  max-width: 140px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 2px;
  background: transparent;
  color: ${({ theme }) => theme.uiForeground};
`

export const Error = styled.div`
  padding: 8px 12px;
  margin: 8px 0;
  color: #b32020;
  font-size: 13px;
  background: #ffeeee;
`

export const Dialog = styled.form`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 320px;

  h1 {
    margin: 0 0 4px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  input[type='text'],
  input:not([type]) {
    width: 100%;
    box-sizing: border-box;
  }
`
