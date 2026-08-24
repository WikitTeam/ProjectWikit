import { t } from '~util/i18n'
import * as React from 'react'
import Trans from '~util/trans'
import { useEffect, useMemo, useRef, useState } from 'react'
import { UserData } from '~api/user'
import {
  canSendMessage,
  DirectMessage,
  getConversation,
  reportMessages,
  sendMessage,
} from '~api/messages'
import { useConfigContext } from '~reactive/config'
import formatDate from '~util/date-format'
import { Paths } from '~reactive/paths'
import WikidotModal, {
  addUnmanagedModal,
  removeUnmanagedModal,
  showErrorModal,
} from '~util/wikidot-modal'
import * as Styled from './Messages.styles'

interface Props {
  partnerId: number
  onMessageSent: () => void
}

const POLL_INTERVAL_MS = 3000

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

  const [selectMode, setSelectMode] = useState<boolean>(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  const listRef = useRef<HTMLDivElement | null>(null)
  const wasAtBottomRef = useRef<boolean>(true)
  const messagesRef = useRef<DirectMessage[]>([])
  const selectModeRef = useRef<boolean>(false)
  const partnerIdRef = useRef<number>(partnerId)

  useEffect(() => { messagesRef.current = messages }, [messages])
  useEffect(() => { selectModeRef.current = selectMode }, [selectMode])
  useEffect(() => { partnerIdRef.current = partnerId }, [partnerId])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setMessages([])
    setPartner(null)
    setSelectMode(false)
    setSelectedIds(new Set())
    wasAtBottomRef.current = true

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
        setError(err?.error || t('messages.conversation-view.load-failed'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [partnerId])

  useEffect(() => {
    if (selectMode) return
    if (!wasAtBottomRef.current) return
    const raf = requestAnimationFrame(() => {
      const el = listRef.current
      if (!el) return
      el.scrollTop = el.scrollHeight
    })
    return () => cancelAnimationFrame(raf)
  }, [messages.length, selectMode, loading])

  const handleScroll = () => {
    const el = listRef.current
    if (!el) return
    wasAtBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40
  }

  useEffect(() => {
    if (!partnerId || Number.isNaN(partnerId)) return

    const poll = async () => {
      if (document.visibilityState !== 'visible') return
      if (selectModeRef.current) return
      if (partnerIdRef.current !== partnerId) return
      try {
        const conv = await getConversation(partnerId, -1, 50, true)
        if (partnerIdRef.current !== partnerId) return
        if (!conv.messages) return
        const freshOrdered = conv.messages.slice().reverse()
        setMessages(prev => {
          const existing = new Set(prev.map(m => m.id))
          const news = freshOrdered.filter(m => !existing.has(m.id))
          if (news.length === 0) return prev
          return [...prev, ...news]
        })
      } catch {
      }
    }

    poll()
    const interval = setInterval(poll, POLL_INTERVAL_MS)
    const onVisibility = () => {
      if (document.visibilityState === 'visible') poll()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      clearInterval(interval)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [partnerId])

  const handleSend = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    const body = draft.trim()
    if (!body || sending) return

    setSending(true)
    setSendError(null)
    try {
      const created = await sendMessage(partnerId, body)
      wasAtBottomRef.current = true
      setMessages(prev => [...prev, created])
      setDraft('')
      onMessageSent()
    } catch (err: any) {
      setSendError(err?.error || t('messages.conversation-view.send-failed'))
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

  const enterSelectMode = () => {
    setSelectMode(true)
    setSelectedIds(new Set())
  }

  const cancelSelectMode = () => {
    setSelectMode(false)
    setSelectedIds(new Set())
  }

  const toggleSelected = (id: number) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const selectAll = () => {
    setSelectedIds(new Set(messages.map(m => m.id)))
  }

  const openReportModal = () => {
    if (selectedIds.size === 0) return
    const ids = Array.from(selectedIds)
    showReportModal(partnerId, ids, () => {
      setSelectMode(false)
      setSelectedIds(new Set())
    })
  }

  const partnerLabel = useMemo(() => {
    if (!partner) return t('messages.conversation-view.unknown-user', { id: partnerId })
    return partner.name
  }, [partner, partnerId])

  if (loading) return <Styled.LoadingBanner>{t('messages.conversation-view.loading')}</Styled.LoadingBanner>
  if (error) return <Styled.ErrorBanner>{error}</Styled.ErrorBanner>

  return (
    <>
      <Styled.ConversationHeader>
        {selectMode ? (
          <Styled.SelectModeToolbar>
            <span>{t('messages.conversation-view.selected-count', { count: selectedIds.size })}</span>
            <Styled.ToolbarAction onClick={selectAll}>{t('messages.conversation-view.select-all')}</Styled.ToolbarAction>
            <Styled.ToolbarAction onClick={cancelSelectMode}>{t('messages.conversation-view.select-cancel')}</Styled.ToolbarAction>
            <Styled.HeaderSpacer />
            <Styled.ToolbarAction danger disabled={selectedIds.size === 0} onClick={openReportModal}>
              {t('messages.conversation-view.next')}
            </Styled.ToolbarAction>
          </Styled.SelectModeToolbar>
        ) : (
          <>
            <Styled.BackButton href={`/-${Paths.messages}`}>← </Styled.BackButton>
            <Trans
              id="messages.conversation-view.with-partner"
              children={{ partner: <a href={`/-/users/${partnerId}-${partner?.username || ''}`}>{partnerLabel}</a> }}
            />
            <Styled.HeaderSpacer />
            <Styled.ReportButton onClick={enterSelectMode}>{t('messages.conversation-view.report')}</Styled.ReportButton>
          </>
        )}
      </Styled.ConversationHeader>
      <Styled.MessageList ref={listRef} onScroll={handleScroll}>
        {messages.length === 0 && (
          <Styled.EmptyState>{t('messages.conversation-view.empty')}</Styled.EmptyState>
        )}
        {messages.map(msg => {
          const mine = msg.sender_id === currentUserId
          const isSelected = selectedIds.has(msg.id)
          const rowContent = (
            <>
              <Styled.MessageRow mine={mine}>
                <Styled.MessageBubble mine={mine}>{msg.body}</Styled.MessageBubble>
              </Styled.MessageRow>
              <Styled.MessageMeta mine={mine}>
                {formatDate(new Date(msg.created_at))}
              </Styled.MessageMeta>
            </>
          )
          if (selectMode) {
            return (
              <Styled.SelectableRow
                key={msg.id}
                selected={isSelected}
                onClick={() => toggleSelected(msg.id)}
              >
                <Styled.MessageCheckbox
                  checked={isSelected}
                  onChange={() => toggleSelected(msg.id)}
                  onClick={e => e.stopPropagation()}
                />
                <div style={{ flex: 1, minWidth: 0 }}>{rowContent}</div>
              </Styled.SelectableRow>
            )
          }
          return <div key={msg.id}>{rowContent}</div>
        })}
      </Styled.MessageList>
      {!canSend && cannotSendReason && !selectMode && (
        <Styled.ErrorBanner>{cannotSendReason}</Styled.ErrorBanner>
      )}
      {sendError && <Styled.ErrorBanner>{sendError}</Styled.ErrorBanner>}
      {!selectMode && (
        <Styled.Composer onSubmit={handleSend}>
          <Styled.ComposerInput
            value={draft}
            onChange={e => setDraft(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={canSend ? t('messages.conversation-view.input-placeholder') : t('messages.conversation-view.cannot-send')}
            disabled={!canSend || sending}
          />
          <Styled.SendButton type="submit" disabled={!canSend || sending || !draft.trim()}>
            {sending ? t('messages.conversation-view.sending') : t('messages.conversation-view.send')}
          </Styled.SendButton>
        </Styled.Composer>
      )}
    </>
  )
}

function showReportModal(reportedId: number, messageIds: number[], onSuccess: () => void) {
  let uuid: string | null = null
  let reason = ''
  let submitting = false

  const close = () => {
    if (uuid) removeUnmanagedModal(uuid)
  }

  const submit = async () => {
    const trimmed = reason.trim()
    if (!trimmed || submitting) return
    submitting = true
    try {
      await reportMessages(reportedId, messageIds, trimmed)
      close()
      onSuccess()
      showInfoModal(t('messages.conversation-view.report-submitted'))
    } catch (err: any) {
      showErrorModal(err?.error || t('messages.conversation-view.report-failed'))
    } finally {
      submitting = false
    }
  }

  uuid = addUnmanagedModal(
    <WikidotModal
      buttons={[
        { title: t('messages.conversation-view.report-cancel'), onClick: close },
        { title: t('messages.conversation-view.report-submit'), onClick: submit, type: 'danger' },
      ]}
    >
      <p>
        <strong>{t('messages.conversation-view.report-count', { count: messageIds.length })}</strong>
      </p>
      <Styled.ReportModalTextarea
        placeholder={t('messages.conversation-view.report-placeholder')}
        onChange={e => {
          reason = e.target.value
        }}
      />
      <Styled.ReportModalHint>
        {t('messages.conversation-view.report-note')}
      </Styled.ReportModalHint>
    </WikidotModal>,
  )
}

function showInfoModal(text: string) {
  let uuid: string | null = null
  const close = () => {
    if (uuid) removeUnmanagedModal(uuid)
  }
  uuid = addUnmanagedModal(
    <WikidotModal buttons={[{ title: t('messages.conversation-view.ok'), onClick: close }]}>
      <p>{text}</p>
    </WikidotModal>,
  )
}

export default ConversationView
