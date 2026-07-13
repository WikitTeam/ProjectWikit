import * as React from 'react'
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

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setMessages([])
    setPartner(null)
    setSelectMode(false)
    setSelectedIds(new Set())

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
    if (listRef.current && !selectMode) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages.length, selectMode])

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
    if (!partner) return `用户 #${partnerId}`
    return partner.name
  }, [partner, partnerId])

  if (loading) return <Styled.LoadingBanner>加载中…</Styled.LoadingBanner>
  if (error) return <Styled.ErrorBanner>{error}</Styled.ErrorBanner>

  return (
    <>
      <Styled.ConversationHeader>
        {selectMode ? (
          <Styled.SelectModeToolbar>
            <span>已选 {selectedIds.size} 条</span>
            <Styled.ToolbarAction onClick={selectAll}>全选</Styled.ToolbarAction>
            <Styled.ToolbarAction onClick={cancelSelectMode}>取消</Styled.ToolbarAction>
            <Styled.HeaderSpacer />
            <Styled.ToolbarAction danger disabled={selectedIds.size === 0} onClick={openReportModal}>
              下一步 →
            </Styled.ToolbarAction>
          </Styled.SelectModeToolbar>
        ) : (
          <>
            <Styled.BackButton href={`/-${Paths.messages}`}>← </Styled.BackButton>
            与 <a href={`/-/users/${partnerId}-${partner?.username || ''}`}>{partnerLabel}</a> 的对话
            <Styled.HeaderSpacer />
            <Styled.ReportButton onClick={enterSelectMode}>检举</Styled.ReportButton>
          </>
        )}
      </Styled.ConversationHeader>
      <Styled.MessageList ref={listRef}>
        {messages.length === 0 && (
          <Styled.EmptyState>还没有消息。发送第一条消息开始对话吧。</Styled.EmptyState>
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
            placeholder={canSend ? '输入消息…（回车发送，Shift+回车换行）' : '你无法向该用户发送消息'}
            disabled={!canSend || sending}
          />
          <Styled.SendButton type="submit" disabled={!canSend || sending || !draft.trim()}>
            {sending ? '发送中…' : '发送'}
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
      showInfoModal('检举已提交，管理员会尽快处理。')
    } catch (err: any) {
      showErrorModal(err?.error || '提交失败')
    } finally {
      submitting = false
    }
  }

  uuid = addUnmanagedModal(
    <WikidotModal
      buttons={[
        { title: '取消', onClick: close },
        { title: '提交检举', onClick: submit, type: 'danger' },
      ]}
    >
      <p>
        <strong>检举 {messageIds.length} 条消息</strong>
      </p>
      <Styled.ReportModalTextarea
        placeholder="请说明检举理由（管理员会看到，最多 2000 字）"
        onChange={e => {
          reason = e.target.value
        }}
      />
      <Styled.ReportModalHint>
        提交后，选中的消息内容会连同你的理由一起发送给管理员。
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
    <WikidotModal buttons={[{ title: '好', onClick: close }]}>
      <p>{text}</p>
    </WikidotModal>,
  )
}

export default ConversationView
