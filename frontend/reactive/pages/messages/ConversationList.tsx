import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { clearConversations, ConversationSummary, getConversations } from '~api/messages'
import { Paths } from '~reactive/paths'
import formatDate from '~util/date-format'
import { t } from '~util/i18n'
import { showConfirmModal, showErrorModal } from '~util/wikidot-modal'
import * as Styled from './Messages.styles'

const POLL_INTERVAL_MS = 3000

interface Props {
  selectedPartnerId: number | null
  refreshToken: number
}

const ConversationList: React.FC<Props> = ({ selectedPartnerId, refreshToken }) => {
  const [conversations, setConversations] = useState<ConversationSummary[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [clearing, setClearing] = useState<boolean>(false)
  const navigate = useNavigate()

  const loadedRef = useRef<boolean>(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getConversations()
      .then(response => {
        if (cancelled) return
        setConversations(response.conversations)
        loadedRef.current = true
      })
      .catch(err => {
        if (cancelled) return
        setError(err?.error || t('messages.conversation-list.load-failed'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [refreshToken])

  useEffect(() => {
    const poll = async () => {
      if (document.visibilityState !== 'visible') return
      if (!loadedRef.current) return
      try {
        const response = await getConversations()
        setConversations(prev => {
          if (sameConversations(prev, response.conversations)) return prev
          return response.conversations
        })
      } catch {}
    }

    const interval = setInterval(poll, POLL_INTERVAL_MS)
    const onVisibility = () => {
      if (document.visibilityState === 'visible') poll()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      clearInterval(interval)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [])

  const toggle = (id: number) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    setSelected(prev => (prev.size === conversations.length ? new Set<number>() : new Set(conversations.map(c => c.partner.id ?? 0))))
  }

  const clearSelected = async () => {
    if (selected.size === 0 || clearing) return
    setClearing(true)
    try {
      const response = await clearConversations(Array.from(selected))
      const gone = new Set(response.cleared)
      setConversations(prev => prev.filter(c => !gone.has(c.partner.id ?? 0)))
      setSelected(new Set())
      if (selectedPartnerId !== null && gone.has(selectedPartnerId)) navigate(Paths.messages)
    } catch (err: any) {
      showErrorModal(err?.error || t('messages.conversation-list.delete-failed'))
    } finally {
      setClearing(false)
    }
  }

  const confirmClear = () => {
    if (selected.size === 0 || clearing) return
    showConfirmModal(
      t('messages.conversation-list.delete-confirm', { count: selected.size }),
      t('messages.conversation-list.delete-confirm-button'),
      clearSelected,
    )
  }

  if (loading) return <Styled.LoadingBanner>{t('messages.conversation-list.loading')}</Styled.LoadingBanner>
  if (error) return <Styled.ErrorBanner>{error}</Styled.ErrorBanner>
  if (conversations.length === 0) {
    return <Styled.LoadingBanner>{t('messages.conversation-list.empty')}</Styled.LoadingBanner>
  }

  return (
    <>
      <Styled.ConversationToolbar>
        <Styled.ConversationToolbarLabel>
          <input type="checkbox" checked={selected.size === conversations.length} onChange={toggleAll} />
          {t('messages.conversation-list.select-all')}
        </Styled.ConversationToolbarLabel>
        <span>{t('messages.conversation-list.selected-count', { count: selected.size })}</span>
        <Styled.HeaderSpacer />
        <Styled.ToolbarAction danger disabled={selected.size === 0 || clearing} onClick={confirmClear}>
          {t('messages.conversation-list.delete-selected')}
        </Styled.ToolbarAction>
      </Styled.ConversationToolbar>
      {conversations.map(conv => {
        const isActive = selectedPartnerId === conv.partner.id
        const isUnread = conv.unread_count > 0
        return (
          <Styled.ConversationSelectRow key={conv.partner.id}>
            <Styled.MessageCheckbox checked={selected.has(conv.partner.id ?? 0)} onChange={() => toggle(conv.partner.id ?? 0)} />
            <Styled.ConversationItem href={`/-${Paths.messages}/${conv.partner.id}`} active={isActive} unread={isUnread}>
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
          </Styled.ConversationSelectRow>
        )
      })}
    </>
  )
}

function sameConversations(a: ConversationSummary[], b: ConversationSummary[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i].partner.id !== b[i].partner.id) return false
    if (a[i].unread_count !== b[i].unread_count) return false
    if (a[i].last_message.id !== b[i].last_message.id) return false
  }
  return true
}

export default ConversationList
