import React, { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArticleLogEntry } from '~api/articles'
import { Notification as INotification, setMute } from '~api/notifications'
import { renderArticleHistoryComment, renderArticleHistoryFlags } from '~articles/article-history'
import { Paths } from '~reactive/paths'
import formatDate from '~util/date-format'
import { t } from '~util/i18n'
import Trans from '~util/trans'
import UserView from '~util/user-view'
import * as Styled from './Notification.styles'

interface Props {
  notification: INotification
  full?: boolean
}

// Long notices are cut to a few lines in the list. Only one that was actually
// cut offers the way to the whole thing.
const Clamped: React.FC<{ full?: boolean; id: number; children: React.ReactNode }> = ({ full, id, children }) => {
  const ref = useRef<HTMLDivElement>(null)
  const [cut, setCut] = useState(false)
  useLayoutEffect(() => {
    if (!full && ref.current) setCut(ref.current.scrollHeight > ref.current.clientHeight + 1)
  }, [full, children])
  if (full) return <>{children}</>
  return (
    <>
      <Styled.Clamp ref={ref} cut={cut}>
        {children}
      </Styled.Clamp>
      {cut && (
        <Styled.ReadMore as={Link} to={`${Paths.notifications}/view/${id}`}>
          {t('notifications.item.read-more')}
        </Styled.ReadMore>
      )}
    </>
  )
}

type Target = { article_id?: number; thread_id?: number; user_id?: number }

function placeTarget(n: INotification): Target | null {
  if (n.type === 'new_article_revision') return { article_id: n.article.uid }
  if (n.type === 'new_post_reply' || n.type === 'new_thread_post' || n.type === 'forum_mention') return { thread_id: n.thread.id }
  return null
}

function actorTarget(n: INotification): Target | null {
  const actor =
    n.type === 'new_article_revision'
      ? n.user
      : n.type === 'new_post_reply' || n.type === 'new_thread_post' || n.type === 'forum_mention' || n.type === 'post_like'
        ? n.author
        : null
  return actor?.id && actor.type !== 'system' ? { user_id: actor.id } : null
}

const MuteButton: React.FC<{ target: Target | null }> = ({ target }) => {
  const [muted, setMuted] = useState(false)
  const [busy, setBusy] = useState(false)
  if (!target) return null
  const onClick = (e: React.MouseEvent) => {
    e.preventDefault()
    if (busy || muted) return
    setBusy(true)
    setMute(target, true)
      .then(() => setMuted(true))
      .finally(() => setBusy(false))
  }
  const label = muted
    ? t('notifications.item.muted')
    : target.article_id
      ? t('notifications.item.mute-page')
      : target.thread_id
        ? t('notifications.item.mute-thread')
        : t('notifications.item.mute-user')
  return (
    <Styled.Mute href="#" onClick={onClick} aria-disabled={muted || busy}>
      {label}
    </Styled.Mute>
  )
}

const Notification: React.FC<Props> = ({ notification, full }) => {
  const unread = !notification.is_viewed
  const base = notification.site?.url ?? ''

  const body = useMemo(() => {
    if (notification.type === 'new_post_reply' || notification.type === 'new_thread_post' || notification.type === 'forum_mention') {
      const title = {
        new_post_reply: t('notifications.item.type-post-reply'),
        new_thread_post: t('notifications.item.type-new-post'),
        forum_mention: t('notifications.item.type-mention'),
      }
      return (
        <>
          <Styled.TypeName>{title[notification.type as keyof typeof title]}</Styled.TypeName>
          <Styled.PostFrom>
            <Trans
              id="notifications.item.from-author-in"
              children={{
                author: <UserView data={notification.author} />,
                section: <a href={base + notification.section.url}>{notification.section.name}</a>,
              }}
            />{' '}
            &raquo; <a href={base + notification.category.url}>{notification.category.name}</a> &raquo;{' '}
            <a href={base + notification.thread.url}>{notification.thread.name}</a>
          </Styled.PostFrom>
          <Styled.PostName>
            <a href={base + notification.post.url}>{notification.post.name || t('notifications.item.view-post')}</a>
          </Styled.PostName>
          <Styled.PostContent>
            <div dangerouslySetInnerHTML={{ __html: notification.message }} />
          </Styled.PostContent>
        </>
      )
    } else if (notification.type === 'new_article_revision') {
      const logEntry: ArticleLogEntry = {
        comment: notification.comment,
        createdAt: notification.created_at,
        meta: notification.rev_meta,
        revNumber: notification.rev_number,
        type: notification.rev_type,
        user: notification.user,
        defaultComment: '',
      }

      const pageName = notification.article.pageId.indexOf(':')
        ? `${notification.article.pageId.split(':')[0]}: ${notification.article.title}`
        : notification.article.title
      const comment = renderArticleHistoryComment(logEntry)

      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-watched-edit')}</Styled.TypeName>
          <Styled.RevisionFields>
            <Styled.RevisionArticle>
              <a href={`${base}/${notification.article.pageId}`}>{pageName}</a>
            </Styled.RevisionArticle>
            <Styled.RevisionFlags>{renderArticleHistoryFlags(logEntry)}</Styled.RevisionFlags>
            <Styled.RevisionNumber>rev.{notification.rev_number}</Styled.RevisionNumber>
            <Styled.RevisionUser>
              <UserView data={notification.user} />
            </Styled.RevisionUser>
          </Styled.RevisionFields>
          {comment && (
            <Styled.RevisionComment>
              <Styled.RevisionCommentCaption>{t('notifications.item.comment-label')}</Styled.RevisionCommentCaption> {comment}
            </Styled.RevisionComment>
          )}
        </>
      )
    } else if (notification.type === 'welcome') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-welcome')}</Styled.TypeName>
          {notification.message && (
            <Styled.PostContent>
              <div dangerouslySetInnerHTML={{ __html: notification.message }} />
            </Styled.PostContent>
          )}
        </>
      )
    } else if (notification.type === 'announcement') {
      return (
        <>
          <Styled.TypeName>
            {notification.title ||
              (notification.instance ? t('notifications.item.type-instance-announcement') : t('notifications.item.type-announcement'))}
          </Styled.TypeName>
          <Styled.PostContent>
            <div dangerouslySetInnerHTML={{ __html: notification.message }} />
          </Styled.PostContent>
        </>
      )
    } else if (notification.type === 'external') {
      return (
        <>
          <Styled.TypeName>{notification.title}</Styled.TypeName>
          {notification.message && (
            <Styled.PostContent>
              <div dangerouslySetInnerHTML={{ __html: notification.message }} />
            </Styled.PostContent>
          )}
          {notification.link && (
            <Styled.PostName>
              <a href={base + notification.link}>{t('notifications.item.open-link')}</a>
            </Styled.PostName>
          )}
        </>
      )
    } else if (notification.type === 'page_deleted') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-page-deleted')}</Styled.TypeName>
          <Styled.PostContent>
            <div dangerouslySetInnerHTML={{ __html: notification.message }} />
          </Styled.PostContent>
        </>
      )
    } else if (notification.type === 'post_like') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-post-like')}</Styled.TypeName>
          <Styled.PostFrom>
            <Trans id="notifications.item.liked-by" children={{ author: <UserView data={notification.author} /> }} /> &raquo;{' '}
            <a href={base + notification.thread.url}>{notification.thread.name}</a>
          </Styled.PostFrom>
          <Styled.PostName>
            <a href={base + notification.post.url}>{notification.post.name || t('notifications.item.view-post')}</a>
          </Styled.PostName>
        </>
      )
    } else if (notification.type === 'release_available') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-release', { version: notification.version })}</Styled.TypeName>
          <Styled.PostContent>{t('notifications.item.release-hint')}</Styled.PostContent>
          {notification.notes && (
            <Styled.PostName>
              <a href={notification.notes} target="_blank" rel="noopener noreferrer">
                {t('notifications.item.release-notes')}
              </a>
            </Styled.PostName>
          )}
        </>
      )
    } else if (notification.type === 'update_rolled_back') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-rolled-back', { version: notification.version, from: notification.from })}</Styled.TypeName>
          <Styled.PostContent>{t('notifications.item.rolled-back-hint')}</Styled.PostContent>
        </>
      )
    } else if (notification.type === 'new_ticket') {
      const path = { ticket: 'tickets', membershipapply: 'membership-applications', report: 'reports' }[notification.kind]
      return (
        <>
          <Styled.TypeName>
            {t(`notifications.item.new-${notification.kind}`, { sender: notification.sender_name, subject: notification.subject })}
          </Styled.TypeName>
          {notification.kind === 'ticket' && notification.subject && <Styled.PostContent>{notification.subject}</Styled.PostContent>}
          <Styled.PostName>
            <a href={`${base}/-/admin/${path}/${notification.ticket_id}`}>{t('notifications.item.view-ticket')}</a>
          </Styled.PostName>
        </>
      )
    } else if (notification.type === 'ticket_result') {
      return (
        <>
          <Styled.TypeName>
            {t(`notifications.item.result-${notification.kind}`, {
              subject: notification.subject,
              status: t(`own-tickets.status-${notification.status}`),
            })}
          </Styled.TypeName>
          {notification.reply && <Styled.PostContent>{notification.reply}</Styled.PostContent>}
          <Styled.PostName>
            <a href="/-/my-tickets">{t('notifications.item.view-own-tickets')}</a>
          </Styled.PostName>
        </>
      )
    } else if (notification.type === 'direct_message') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.new-message', { sender: notification.sender_name })}</Styled.TypeName>
          <Styled.PostContent>{notification.preview}</Styled.PostContent>
          <Styled.PostName>
            <a href={`/-/messages/${notification.sender_id}`}>{t('notifications.item.view-conversation')}</a>
          </Styled.PostName>
        </>
      )
    } else {
      return <Styled.TypeName>{t('notifications.item.render-failed')}</Styled.TypeName>
    }
  }, [notification])

  return (
    <Styled.Container unread={unread}>
      <Styled.TypeMark unread={unread} />
      <Styled.Body>
        {notification.site && (
          <Styled.SiteName>
            <Trans id="notifications.item.from-site" children={{ site: <a href={notification.site.url}>{notification.site.title}</a> }} />
          </Styled.SiteName>
        )}
        <Clamped full={full} id={notification.id}>
          {body}
        </Clamped>
        <Styled.MuteRow>
          <MuteButton target={placeTarget(notification)} />
          <MuteButton target={actorTarget(notification)} />
        </Styled.MuteRow>
      </Styled.Body>
      <Styled.NotificationDate>{formatDate(new Date(notification.created_at))}</Styled.NotificationDate>
    </Styled.Container>
  )
}

export default Notification
