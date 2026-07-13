import { UserData } from '~api/user'
import { wFetch } from '../util/fetch-util'

export interface DirectMessage {
  id: number
  sender_id: number
  recipient_id: number
  body: string
  created_at: string
  is_read: boolean
}

export interface ConversationSummary {
  partner: UserData
  last_message: {
    id: number
    sender_id: number
    preview: string
    created_at: string
  }
  unread_count: number
}

export interface ConversationsResponse {
  conversations: ConversationSummary[]
}

export interface ConversationResponse {
  partner: UserData
  messages: DirectMessage[]
  cursor: number
}

export interface CanSendResponse {
  allowed: boolean
  reason?: string | null
}

export async function getConversations(): Promise<ConversationsResponse> {
  return await wFetch<ConversationsResponse>('/api/messages/conversations')
}

export async function getConversation(
  partnerId: number,
  cursor: number = -1,
  limit: number = 30,
  markRead: boolean = false,
): Promise<ConversationResponse> {
  return await wFetch<ConversationResponse>(
    `/api/messages/with/${partnerId}?cursor=${cursor}&limit=${limit}&mark_read=${markRead}`,
  )
}

export async function sendMessage(recipientId: number, body: string): Promise<DirectMessage> {
  return await wFetch<DirectMessage>('/api/messages/send', {
    method: 'POST',
    sendJson: true,
    body: { recipient_id: recipientId, body },
  })
}

export async function canSendMessage(userId: number): Promise<CanSendResponse> {
  return await wFetch<CanSendResponse>(`/api/messages/can-send/${userId}`)
}

export async function blockUser(userId: number): Promise<{ status: string; blocked: boolean }> {
  return await wFetch(`/api/users/${userId}/block`, { method: 'POST', sendJson: true, body: {} })
}

export async function unblockUser(userId: number): Promise<{ status: string; blocked: boolean }> {
  return await wFetch(`/api/users/${userId}/block`, { method: 'DELETE', sendJson: true, body: {} })
}

export async function reportMessages(
  reportedId: number,
  messageIds: number[],
  reason: string,
): Promise<{ status: string; report_id: number }> {
  return await wFetch('/api/messages/report', {
    method: 'POST',
    sendJson: true,
    body: { reported_id: reportedId, message_ids: messageIds, reason },
  })
}
