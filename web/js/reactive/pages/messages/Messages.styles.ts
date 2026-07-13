import styled, { css } from 'styled-components'

export const Layout = styled.div`
  display: flex;
  min-height: 500px;
  height: calc(100vh - 200px);
  border: 1px solid ${({ theme }) => theme.windowStrong};

  @media (max-width: 700px) {
    flex-direction: column;
    height: auto;
  }
`

export const SidebarSearch = styled.form`
  display: flex;
  gap: 6px;
  padding: 10px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  background: ${({ theme }) => theme.windowPadding};
  position: sticky;
  top: 0;
  z-index: 1;
`

export const SearchInput = styled.input`
  flex: 1;
  min-width: 0;
  padding: 6px 8px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 4px;
  font: inherit;
`

export const SearchButton = styled.button`
  padding: 6px 10px;
  border: none;
  border-radius: 4px;
  background: ${({ theme }) => theme.uiSelection};
  color: ${({ theme }) => theme.uiSelectionForeground};
  cursor: pointer;
  font-weight: 500;
  white-space: nowrap;

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`

export const SearchError = styled.div`
  padding: 6px 10px;
  color: #cc3333;
  font-size: 12px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  background: #ffeeee;
`

export const Sidebar = styled.div<{ hasSelection: boolean }>`
  width: 280px;
  min-width: 240px;
  border-right: 1px solid ${({ theme }) => theme.windowStrong};
  overflow-y: auto;
  background: ${({ theme }) => theme.windowPadding};

  @media (max-width: 700px) {
    width: 100%;
    max-height: 40vh;
    border-right: none;
    border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
    ${({ hasSelection }) =>
      hasSelection &&
      css`
        display: none;
      `};
  }
`

export const Main = styled.div`
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
`

export const EmptyState = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: ${({ theme }) => theme.uiForeground};
  opacity: 0.6;
  padding: 24px;
  text-align: center;
`

export const ConversationItem = styled.a<{ active: boolean; unread: boolean }>`
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  text-decoration: none;
  color: inherit;
  cursor: pointer;

  ${({ active, theme }) =>
    active &&
    css`
      background: ${theme.higlightBackground};
    `};

  &:hover {
    background: ${({ theme }) => theme.higlightBackground};
    text-decoration: none;
  }
`

export const ConversationDot = styled.span<{ unread: boolean }>`
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 8px;
  flex-shrink: 0;
  background: ${({ unread, theme }) => (unread ? theme.uiSelection : 'transparent')};
`

export const ConversationInfo = styled.div`
  flex: 1;
  min-width: 0;
`

export const ConversationName = styled.div<{ unread: boolean }>`
  font-weight: ${({ unread }) => (unread ? 600 : 400)};
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

export const ConversationPreview = styled.div<{ unread: boolean }>`
  font-size: 12px;
  color: ${({ theme }) => theme.uiForeground};
  opacity: ${({ unread }) => (unread ? 0.9 : 0.6)};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
`

export const ConversationMeta = styled.div`
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
  opacity: 0.6;
  text-align: right;
  flex-shrink: 0;
`

export const UnreadBadge = styled.span`
  display: inline-block;
  background: ${({ theme }) => theme.uiSelection};
  color: ${({ theme }) => theme.uiSelectionForeground};
  font-size: 11px;
  line-height: 1;
  padding: 2px 6px;
  border-radius: 10px;
  margin-top: 4px;
`

export const ConversationHeader = styled.div`
  padding: 10px 16px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  font-weight: 500;
  background: ${({ theme }) => theme.windowPadding};
  display: flex;
  align-items: center;
  gap: 8px;
`

export const BackButton = styled.a`
  display: none;
  cursor: pointer;

  @media (max-width: 700px) {
    display: inline;
  }
`

export const HeaderSpacer = styled.div`
  flex: 1;
`

export const ReportButton = styled.a`
  cursor: pointer;
  color: #c33;
  text-decoration: none;
  font-size: 13px;

  &:hover {
    text-decoration: underline;
  }
`

export const SelectModeToolbar = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  font-size: 13px;
`

export const ToolbarAction = styled.a<{ danger?: boolean; disabled?: boolean }>`
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  color: ${({ danger }) => (danger ? '#c33' : 'inherit')};
  opacity: ${({ disabled }) => (disabled ? 0.4 : 1)};
  text-decoration: none;
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};

  &:hover {
    text-decoration: underline;
  }
`

export const MessageCheckbox = styled.input.attrs({ type: 'checkbox' })`
  margin: 0 6px 0 0;
  cursor: pointer;
  flex-shrink: 0;
  align-self: center;
`

export const SelectableRow = styled.div<{ selected: boolean }>`
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  background: ${({ selected, theme }) => (selected ? theme.higlightBackground : 'transparent')};

  &:hover {
    background: ${({ theme }) => theme.higlightBackground};
  }
`

export const ReportModalTextarea = styled.textarea`
  width: 100%;
  min-height: 100px;
  max-height: 300px;
  resize: vertical;
  padding: 8px;
  border: 1px solid #aaaaaa;
  border-radius: 4px;
  font: inherit;
  box-sizing: border-box;
`

export const ReportModalHint = styled.div`
  font-size: 12px;
  color: #666666;
  margin-top: 6px;
`

export const MessageList = styled.div`
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
`

export const MessageRow = styled.div<{ mine: boolean }>`
  display: flex;
  justify-content: ${({ mine }) => (mine ? 'flex-end' : 'flex-start')};
`

export const MessageBubble = styled.div<{ mine: boolean }>`
  max-width: 70%;
  padding: 8px 12px;
  border-radius: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  background: ${({ mine, theme }) => (mine ? theme.uiSelection : theme.higlightBackground)};
  color: ${({ mine, theme }) => (mine ? theme.uiSelectionForeground : theme.foreground)};
`

export const MessageMeta = styled.div<{ mine: boolean }>`
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
  opacity: 0.6;
  margin-top: 2px;
  text-align: ${({ mine }) => (mine ? 'right' : 'left')};
`

export const Composer = styled.form`
  border-top: 1px solid ${({ theme }) => theme.windowStrong};
  padding: 10px;
  display: flex;
  gap: 8px;
  align-items: flex-end;
`

export const ComposerInput = styled.textarea`
  flex: 1;
  min-height: 44px;
  max-height: 160px;
  resize: vertical;
  padding: 8px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 4px;
  font: inherit;
`

export const SendButton = styled.button`
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  background: ${({ theme }) => theme.uiSelection};
  color: ${({ theme }) => theme.uiSelectionForeground};
  cursor: pointer;
  font-weight: 500;

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`

export const ErrorBanner = styled.div`
  background: #ffeeee;
  border: 1px #ffcccc dashed;
  padding: 8px;
  margin: 8px;
  font-size: 13px;
`

export const LoadingBanner = styled.div`
  padding: 8px;
  text-align: center;
  font-size: 13px;
  opacity: 0.7;
`
