import { UserData } from '~api/user'
import { wFetch } from '../util/fetch-util'

interface NotificationEntity {
  id: number
  name: string
  url: string
}

interface BaseNotification {
  id: number
  created_at: string
  is_viewed: boolean
  site?: {
    title: string
    url: string
  }
}

interface NotificationWelcome extends BaseNotification {
  type: 'welcome'
  message?: string
}

interface NotificationNewArticleRevision extends BaseNotification {
  type: 'new_article_revision'
  user: UserData
  article: {
    uid: number
    pageId: string
    title: string
  }
  rev_id: number
  rev_number: number
  rev_type: string
  rev_meta: Record<string, any>
  comment: string
}

interface NotificationNewThreadPost extends BaseNotification {
  type: 'new_thread_post'
  author: UserData
  section: NotificationEntity
  category: NotificationEntity
  thread: NotificationEntity
  post: NotificationEntity
  message: string
}

interface NotificationNewPostReply extends BaseNotification {
  type: 'new_post_reply'
  author: UserData
  section: NotificationEntity
  category: NotificationEntity
  thread: NotificationEntity
  post: NotificationEntity
  origin: NotificationEntity
  message: string
}

interface NotificationForumMention extends BaseNotification {
  type: 'forum_mention'
  author: UserData
  section: NotificationEntity
  category: NotificationEntity
  thread: NotificationEntity
  post: NotificationEntity
  message: string
}

interface NotificationDirectMessage extends BaseNotification {
  type: 'direct_message'
  sender_id: number
  sender_name: string
  message_id: number
  preview: string
}

interface NotificationPostLike extends BaseNotification {
  type: 'post_like'
  author: UserData
  thread: NotificationEntity
  post: NotificationEntity
}

interface NotificationReleaseAvailable extends BaseNotification {
  type: 'release_available'
  version: string
  notes: string
}

interface NotificationUpdateRolledBack extends BaseNotification {
  type: 'update_rolled_back'
  version: string
  from: string
}

interface NotificationNewTicket extends BaseNotification {
  type: 'new_ticket'
  kind: 'ticket' | 'membershipapply' | 'report'
  ticket_id: number
  subject: string
  sender_name: string
}

interface NotificationTicketResult extends BaseNotification {
  type: 'ticket_result'
  kind: 'ticket' | 'membershipapply' | 'report'
  ticket_id: number
  subject: string
  status: string
  reply: string
}

interface NotificationAnnouncement extends BaseNotification {
  type: 'announcement'
  title: string
  message: string
  author: string
  instance: boolean
}

interface NotificationPageDeleted extends BaseNotification {
  type: 'page_deleted'
  title: string
  page: string
  message: string
}

interface NotificationExternal extends BaseNotification {
  type: 'external'
  title: string
  message: string
  link: string
  from: string
}

export type Notification =
  | NotificationExternal
  | NotificationNewPostReply
  | NotificationNewThreadPost
  | NotificationWelcome
  | NotificationNewArticleRevision
  | NotificationForumMention
  | NotificationDirectMessage
  | NotificationPostLike
  | NotificationReleaseAvailable
  | NotificationUpdateRolledBack
  | NotificationNewTicket
  | NotificationTicketResult
  | NotificationPageDeleted
  | NotificationAnnouncement

export interface NotificationsResponse {
  cursor: number
  notifications: Notification[]
}

export interface NotificationSubscriptionData {
  pageId?: string
  forumThreadId?: number
}

export interface NotificationSubscriptionResponse {
  status?: string
}

export interface UnreadCounts {
  notices: number
  comments: number
  messages: number
}

export type NotificationKind = 'notices' | 'new_ticket' | 'comments' | 'replies' | 'forum_mention' | 'post_like'

// The reply tab covers both shapes a forum answer can arrive as, so one tab
// does not leave half of them behind.
const KIND_QUERY: Record<NotificationKind, string> = {
  notices: 'notices',
  new_ticket: 'new_ticket,ticket_result',
  comments: 'comments',
  replies: 'new_post_reply,new_thread_post',
  forum_mention: 'forum_mention',
  post_like: 'post_like',
}

export async function getNotifications(
  cursor: number,
  limit: number = 10,
  unread: boolean = false,
  mark_viewed: boolean = false,
  kind: NotificationKind = 'notices',
) {
  const type = KIND_QUERY[kind] ? `&type=${KIND_QUERY[kind]}` : ''
  return await wFetch<NotificationsResponse>(
    `/pw-api/notifications?cursor=${cursor}&limit=${limit}&unread=${unread}&mark_as_viewed=${mark_viewed}${type}`,
  )
}

export async function setMute(target: { article_id?: number; thread_id?: number; user_id?: number }, muted: boolean) {
  return await wFetch<{ status: string }>(`/pw-api/notifications/mute`, { method: muted ? 'POST' : 'DELETE', sendJson: true, body: target })
}

export async function getNotification(id: number) {
  return await wFetch<NotificationsResponse>(`/pw-api/notifications?id=${id}&mark_as_viewed=true`)
}

export async function clearNotifications(ids: number[]) {
  return await wFetch<{ removed: number }>(`/pw-api/notifications`, { method: 'DELETE', sendJson: true, body: { ids } })
}

export async function clearAllNotifications(kind: NotificationKind) {
  return await wFetch<{ removed: number }>(`/pw-api/notifications`, {
    method: 'DELETE',
    sendJson: true,
    body: { all: true, type: KIND_QUERY[kind] || undefined },
  })
}

export async function subscribeToNotifications(data: NotificationSubscriptionData) {
  return await wFetch<NotificationSubscriptionResponse>(`/pw-api/notifications/subscribe`, { method: 'POST', sendJson: true, body: data })
}

export async function unsubscribeFromNotifications(data: NotificationSubscriptionData) {
  return await wFetch<NotificationSubscriptionResponse>(`/pw-api/notifications/subscribe`, { method: 'DELETE', sendJson: true, body: data })
}
