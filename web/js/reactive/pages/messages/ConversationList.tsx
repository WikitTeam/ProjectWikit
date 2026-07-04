import * as React from 'react'
import { useEffect, useState } from 'react'
import { ConversationSummary, getConversations } from '~api/messages'
import formatDate from '~util/date-format'
import { Paths } from '~reactive/paths'
import * as Styled from './Messages.styles'

interface Props {
  selectedPartnerId: number | null
  refreshToken: number
}

const ConversationList: React.FC<Props> = ({ selectedPartnerId, refreshToken }) => {
  const [conversations, setConversations] = useState<ConversationSummary[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getConversations()
      .then(response => {
        if (cancelled) return
        setConversations(response.conversations)
      })
      .catch(err => {
        if (cancelled) return
        setError(err?.error || '加载会话列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [refreshToken])

  if (loading) return <Styled.LoadingBanner>加载中…</Styled.LoadingBanner>
  if (error) return <Styled.ErrorBanner>{error}</Styled.ErrorBanner>
  if (conversations.length === 0) {
    return <Styled.LoadingBanner>暂无会话。前往任意用户资料页可发起私信。</Styled.LoadingBanner>
  }

  return (
    <>
      {conversations.map(conv => {
        const isActive = selectedPartnerId === conv.partner.id
        const isUnread = conv.unread_count > 0
        return (
          <Styled.ConversationItem
            key={conv.partner.id}
            href={`/-${Paths.messages}/${conv.partner.id}`}
            active={isActive}
            unread={isUnread}
          >
            <Styled.ConversationDot unread={isUnread} />
            <Styled.ConversationInfo>
              <Styled.ConversationName unread={isUnread}>{conv.partner.name}</Styled.ConversationName>
              <Styled.ConversationPreview unread={isUnread}>{conv.last_message.preview}</Styled.ConversationPreview>
            </Styled.ConversationInfo>
            <Styled.ConversationMeta>
              <div>{formatDate(new Date(conv.last_message.created_at))}</div>
              {isUnread && <Styled.UnreadBadge>{conv.unread_count}</Styled.UnreadBadge>}
            </Styled.ConversationMeta>
          </Styled.ConversationItem>
        )
      })}
    </>
  )
}

export default ConversationList
