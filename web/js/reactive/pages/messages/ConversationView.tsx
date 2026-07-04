import * as React from 'react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { UserData } from '~api/user'
import {
  canSendMessage,
  DirectMessage,
  getConversation,
  sendMessage,
} from '~api/messages'
import { useConfigContext } from '~reactive/config'
import formatDate from '~util/date-format'
import { Paths } from '~reactive/paths'
import * as Styled from './Messages.styles'

interface Props {
  partnerId: number
  onMessageSent: () => void
}

const ConversationView: React.FC<Props> = ({ partnerId, onMessageSent }) => {
  const config = useConfigContext()
  const currentUserId = config.user.id

  const [partner, setPartner] = useState<UserData | null>(null)
  const [messages, setMessages] = useState<DirectMessage[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [canSend, setCanSend] = useState<boolean>(true)
  const [cannotSendReason, setCannotSendReason] = useState<string | null>(null)

  const [draft, setDraft] = useState<string>('')
  const [sending, setSending] = useState<boolean>(false)
  const [sendError, setSendError] = useState<string | null>(null)

  const listRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setMessages([])
    setPartner(null)

    Promise.all([
      getConversation(partnerId, -1, 50, true),
      canSendMessage(partnerId),
    ])
      .then(([conv, perm]) => {
        if (cancelled) return
        setPartner(conv.partner)
        setMessages(conv.messages.slice().reverse())
        setCanSend(perm.allowed)
        setCannotSendReason(perm.reason || null)
      })
      .catch(err => {
        if (cancelled) return
        setError(err?.error || '加载会话失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [partnerId])

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages.length])

  const handleSend = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    const body = draft.trim()
    if (!body || sending) return

    setSending(true)
    setSendError(null)
    try {
      const created = await sendMessage(partnerId, body)
      setMessages(prev => [...prev, created])
      setDraft('')
      onMessageSent()
    } catch (err: any) {
      setSendError(err?.error || '发送失败')
    } finally {
      setSending(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const partnerLabel = useMemo(() => {
    if (!partner) return `用户 #${partnerId}`
    return partner.name
  }, [partner, partnerId])

  if (loading) return <Styled.LoadingBanner>加载中…</Styled.LoadingBanner>
  if (error) return <Styled.ErrorBanner>{error}</Styled.ErrorBanner>

  return (
    <>
      <Styled.ConversationHeader>
        <Styled.BackButton href={`/-${Paths.messages}`}>← </Styled.BackButton>
        与 <a href={`/-/users/${partnerId}-${partner?.username || ''}`}>{partnerLabel}</a> 的对话
      </Styled.ConversationHeader>
      <Styled.MessageList ref={listRef}>
        {messages.length === 0 && (
          <Styled.EmptyState>还没有消息。发送第一条消息开始对话吧。</Styled.EmptyState>
        )}
        {messages.map(msg => {
          const mine = msg.sender_id === currentUserId
          return (
            <div key={msg.id}>
              <Styled.MessageRow mine={mine}>
                <Styled.MessageBubble mine={mine}>{msg.body}</Styled.MessageBubble>
              </Styled.MessageRow>
              <Styled.MessageMeta mine={mine}>
                {formatDate(new Date(msg.created_at))}
              </Styled.MessageMeta>
            </div>
          )
        })}
      </Styled.MessageList>
      {!canSend && cannotSendReason && (
        <Styled.ErrorBanner>{cannotSendReason}</Styled.ErrorBanner>
      )}
      {sendError && <Styled.ErrorBanner>{sendError}</Styled.ErrorBanner>}
      <Styled.Composer onSubmit={handleSend}>
        <Styled.ComposerInput
          value={draft}
          onChange={e => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={canSend ? '输入消息…（回车发送，Shift+回车换行）' : '你无法向该用户发送消息'}
          disabled={!canSend || sending}
        />
        <Styled.SendButton type="submit" disabled={!canSend || sending || !draft.trim()}>
          {sending ? '发送中…' : '发送'}
        </Styled.SendButton>
      </Styled.Composer>
    </>
  )
}

export default ConversationView
