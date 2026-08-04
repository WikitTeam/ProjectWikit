import styled from 'styled-components'

export const Container = styled.div<{ unread?: boolean }>`
  display: grid;
  grid-template-columns: 96px 1fr 80px;
  gap: 16px;
  padding: 16px 0;
  border-top: 1px solid ${({ theme }) => theme.windowStrong};
  align-items: baseline;

  &:last-child {
    border-bottom: 1px solid ${({ theme }) => theme.windowStrong};
  }

  &:hover {
    background: ${({ theme }) => theme.higlightBackground};
  }

  @media (max-width: 720px) {
    grid-template-columns: 1fr;
  }
`

export const TypeMark = styled.div<{ unread?: boolean }>`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
  letter-spacing: 0.02em;
  padding-left: 4px;

  &::before {
    content: '${({ unread }) => (unread ? '● ' : '')}';
    color: ${({ theme }) => theme.foreground};
    font-size: 9px;
    vertical-align: middle;
  }

  @media (max-width: 720px) { padding-left: 0; }
`

export const Body = styled.div`
  min-width: 0;
`

export const TypeName = styled.h2`
  font-size: 15px;
  font-weight: 500;
  color: ${({ theme }) => theme.foreground};
  margin: 0;
  line-height: 1.35;
`

export const PostFrom = styled.div`
  margin-top: 4px;
  font-size: 13px;
  color: ${({ theme }) => theme.uiForeground};

  a {
    color: ${({ theme }) => theme.foreground};
    text-decoration: none;
    &:hover { text-decoration: underline; text-underline-offset: 3px; }
  }
`

export const PostContent = styled.div`
  overflow: hidden;
  margin-top: 8px;
  padding: 8px 10px;
  border-left: 2px solid ${({ theme }) => theme.windowStrong};
  color: ${({ theme }) => theme.uiForeground};
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;

  *, *::before, *::after {
    box-sizing: content-box;
  }
`

export const PostName = styled.div`
  margin-top: 6px;
  font-size: 13px;

  a {
    color: ${({ theme }) => theme.foreground};
    text-decoration: underline;
    text-underline-offset: 3px;
    &:hover { color: ${({ theme }) => theme.foreground}; }
  }
`

export const RevisionFields = styled.div`
  margin-top: 8px;
  padding: 8px 10px;
  border-left: 2px solid ${({ theme }) => theme.windowStrong};
  display: grid;
  grid-template-columns: 1fr max-content max-content max-content;
  gap: 8px;
  font-size: 13px;
`

const RevisionField = styled.div``

export const RevisionArticle = styled(RevisionField)`
  a {
    color: ${({ theme }) => theme.foreground};
    text-decoration: none;
    &:hover { text-decoration: underline; text-underline-offset: 3px; }
  }
`
export const RevisionFlags = styled(RevisionField)`
  color: ${({ theme }) => theme.uiForeground};
`
export const RevisionNumber = styled(RevisionField)`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
`
export const RevisionUser = styled(RevisionField)``

export const RevisionComment = styled.div`
  margin-top: 6px;
  color: ${({ theme }) => theme.uiForeground};
  font-size: 13px;
`

export const RevisionCommentCaption = styled.span`
  font-weight: 500;
  color: ${({ theme }) => theme.foreground};
`

export const NotificationDate = styled.div`
  font-family: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 11px;
  color: ${({ theme }) => theme.uiForeground};
  text-align: right;
  padding-right: 4px;

  @media (max-width: 720px) {
    text-align: left;
    padding-right: 0;
  }
`
