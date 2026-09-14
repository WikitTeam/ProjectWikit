import styled, { css } from 'styled-components'

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

  b { color: ${({ theme }) => theme.foreground}; font-weight: 500; }
  .sep { color: ${({ theme }) => theme.windowStrong}; margin: 0 6px; }
`

export const H1 = styled.h1`
  font-size: 32px;
  font-weight: 500;
  line-height: 1.1;
  letter-spacing: -0.02em;
  margin: 0;
  color: ${({ theme }) => theme.foreground};
`

export const Layout = styled.div`
  display: grid;
  grid-template-columns: 280px 1fr;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 3px;
  min-height: 560px;
  height: calc(100vh - 260px);
  overflow: hidden;

  @media (max-width: 700px) {
    display: flex;
    flex-direction: column;
    height: calc(100vh - 240px);
    height: calc(100dvh - 240px);
    min-height: 400px;
  }
`

export const Sidebar = styled.div<{ hasSelection: boolean }>`
  border-right: 1px solid ${({ theme }) => theme.windowStrong};
  background: ${({ theme }) => theme.windowPadding};
  display: flex;
  flex-direction: column;
  min-height: 0;

  @media (max-width: 700px) {
    max-height: 45vh;
    flex-shrink: 0;
    border-right: none;
    border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
    ${({ hasSelection }) => hasSelection && css`display: none;`};
  }
`

export const ConvListScroll = styled.div`
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
`

export const Main = styled.div`
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: ${({ theme }) => theme.windowPadding};

  @media (max-width: 700px) {
    flex: 1;
  }
`

export const SidebarSearch = styled.form`
  display: flex;
  gap: 6px;
  padding: 10px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  background: ${({ theme }) => theme.windowPadding};
`

export const SearchInput = styled.input`
  flex: 1;
  min-width: 0;
  padding: 7px 10px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 2px;
  outline: none;
  font-size: 13px;
  font-family: inherit;

  &:focus { border-color: ${({ theme }) => theme.foreground}; }
`

export const SearchButton = styled.button`
  padding: 7px 12px;
  background: ${({ theme }) => theme.uiSelection};
  color: ${({ theme }) => theme.uiSelectionForeground};
  border: 1px solid ${({ theme }) => theme.uiSelection};
  border-radius: 2px;
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  cursor: pointer;
  white-space: nowrap;

  &:disabled {
    background: ${({ theme }) => theme.uiForeground};
    border-color: ${({ theme }) => theme.uiForeground};
    cursor: not-allowed;
  }
`

export const SearchError = styled.div`
  padding: 8px 12px;
  color: #b32020;
  font-size: 12px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  background: #ffeeee;
`

export const EmptyState = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: ${({ theme }) => theme.uiForeground};
  padding: 32px 24px;
  text-align: center;
  font-size: 14px;
`

export const ConversationItem = styled.a<{ active: boolean; unread: boolean }>`
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  text-decoration: none;
  color: inherit;
  cursor: pointer;

  ${({ active, theme }) =>
    active &&
    css`
      background: ${theme.higlightBackground};
      box-shadow: inset 2px 0 0 ${theme.foreground};
    `};

  &:hover {
    background: ${({ theme }) => theme.higlightBackground};
    text-decoration: none;
  }
`

export const ConversationDot = styled.span<{ unread: boolean }>`
  display: none;
`

export const ConversationInfo = styled.div`
  flex: 1;
  min-width: 0;
`

export const ConversationName = styled.div<{ unread: boolean }>`
  font-weight: ${({ unread }) => (unread ? 600 : 500)};
  font-size: 14px;
  color: ${({ theme }) => theme.foreground};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &::before {
    content: '${({ unread }) => (unread ? '● ' : '')}';
    color: ${({ theme }) => theme.foreground};
    font-size: 8px;
    vertical-align: middle;
  }
`

export const ConversationPreview = styled.div<{ unread: boolean }>`
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
`

export const ConversationMeta = styled.div`
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
`

export const UnreadBadge = styled.span`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 10px;
  padding: 1px 6px;
  background: ${({ theme }) => theme.uiSelection};
  color: ${({ theme }) => theme.uiSelectionForeground};
  border-radius: 999px;
  line-height: 1.4;
`

export const ConversationHeader = styled.div`
  padding: 12px 20px;
  border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  font-weight: 500;
  background: ${({ theme }) => theme.windowPadding};
  display: flex;
  align-items: center;
  gap: 14px;
  color: ${({ theme }) => theme.foreground};

  a { color: ${({ theme }) => theme.foreground}; text-decoration: none; }
  a:hover { text-decoration: underline; text-underline-offset: 3px; }
`

export const BackButton = styled.a`
  display: none;
  cursor: pointer;
  color: ${({ theme }) => theme.uiForeground};
  text-decoration: none;

  @media (max-width: 700px) { display: inline; }
`

export const HeaderSpacer = styled.div`
  flex: 1;
`

export const ReportButton = styled.a`
  cursor: pointer;
  color: ${({ theme }) => theme.uiForeground};
  text-decoration: none;
  font-size: 13px;

  &:hover { color: #b32020; text-decoration: underline; text-underline-offset: 3px; }
`

export const SelectModeToolbar = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  font-size: 13px;
  color: ${({ theme }) => theme.foreground};
`

export const ToolbarAction = styled.a<{ danger?: boolean; disabled?: boolean }>`
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  color: ${({ danger, theme }) => (danger ? '#b32020' : theme.foreground)};
  opacity: ${({ disabled }) => (disabled ? 0.4 : 1)};
  text-decoration: none;
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};

  &:hover { text-decoration: underline; text-underline-offset: 3px; }
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

  &:hover { background: ${({ theme }) => theme.higlightBackground}; }
`

export const ReportModalTextarea = styled.textarea`
  width: 100%;
  min-height: 100px;
  max-height: 300px;
  resize: vertical;
  padding: 8px 10px;
  border: 1px solid #aaaaaa;
  border-radius: 3px;
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
  padding: 20px 20px 8px;
  display: flex;
  flex-direction: column;
  gap: 10px;
`

export const MessageRow = styled.div<{ mine: boolean }>`
  display: flex;
  justify-content: ${({ mine }) => (mine ? 'flex-end' : 'flex-start')};
`

export const MessageBubble = styled.div<{ mine: boolean }>`
  max-width: 68%;
  padding: 9px 13px;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14.5px;
  line-height: 1.5;

  ${({ mine, theme }) =>
    mine
      ? css`
          background: ${theme.uiSelection};
          color: ${theme.uiSelectionForeground};
          border: 1px solid ${theme.uiSelection};
          border-top-right-radius: 2px;
        `
      : css`
          background: ${theme.windowPadding};
          color: ${theme.foreground};
          border: 1px solid ${theme.windowStrong};
          border-top-left-radius: 2px;
        `}
`

export const MessageMeta = styled.div<{ mine: boolean }>`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 10.5px;
  color: ${({ theme }) => theme.uiForeground};
  padding: 2px 4px 0;
  text-align: ${({ mine }) => (mine ? 'right' : 'left')};
`

export const Composer = styled.form`
  border-top: 1px solid ${({ theme }) => theme.windowStrong};
  padding: 10px 20px 14px;
  display: flex;
  gap: 10px;
  align-items: flex-end;
`

export const ComposerInput = styled.textarea`
  flex: 1;
  min-height: 44px;
  max-height: 160px;
  resize: vertical;
  padding: 10px 12px;
  border: 1px solid ${({ theme }) => theme.windowStrong};
  border-radius: 3px;
  font: inherit;
  font-size: 14px;
  outline: none;

  &:focus { border-color: ${({ theme }) => theme.foreground}; }
`

export const SendButton = styled.button`
  padding: 10px 18px;
  border: 1px solid ${({ theme }) => theme.uiSelection};
  border-radius: 3px;
  background: ${({ theme }) => theme.uiSelection};
  color: ${({ theme }) => theme.uiSelectionForeground};
  cursor: pointer;
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;

  &:disabled {
    background: ${({ theme }) => theme.uiForeground};
    border-color: ${({ theme }) => theme.uiForeground};
    cursor: not-allowed;
  }
`

export const ErrorBanner = styled.div`
  background: transparent;
  border: 1px solid #b32020;
  border-radius: 3px;
  padding: 8px 12px;
  margin: 8px;
  font-size: 13px;
  color: #b32020;
`

export const LoadingBanner = styled.div`
  padding: 16px;
  text-align: center;
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};
`
