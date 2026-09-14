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
}

interface NotificationWelcome extends BaseNotification {
  type: 'welcome'
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

export type Notification =
  | NotificationNewPostReply
  | NotificationNewThreadPost
  | NotificationWelcome
  | NotificationNewArticleRevision
  | NotificationForumMention
  | NotificationDirectMessage
  | NotificationPostLike

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

export type NotificationKind = 'all' | 'post_like' | 'replies' | 'direct_message'

// The reply tab covers both shapes a forum answer can arrive as, so one tab
// does not leave half of them behind.
const KIND_QUERY: Record<NotificationKind, string> = {
  all: '',
  post_like: 'post_like',
  replies: 'new_post_reply,new_thread_post',
  direct_message: 'direct_message',
}

export async function getNotifications(
  cursor: number,
  limit: number = 10,
  unread: boolean = false,
  mark_viewed: boolean = false,
  kind: NotificationKind = 'all',
) {
  const type = KIND_QUERY[kind] ? `&type=${KIND_QUERY[kind]}` : ''
  return await wFetch<NotificationsResponse>(
    `/pw-api/notifications?cursor=${cursor}&limit=${limit}&unread=${unread}&mark_as_viewed=${mark_viewed}${type}`,
  )
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
