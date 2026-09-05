import Trans from '~util/trans'
import { t } from '~util/i18n'
import React, { useMemo } from 'react'
import { ArticleLogEntry } from '~api/articles'
import { Notification as INotification } from '~api/notifications'
import { renderArticleHistoryComment, renderArticleHistoryFlags } from '~articles/article-history'
import formatDate from '~util/date-format'
import UserView from '~util/user-view'
import * as Styled from './Notification.styles'

interface Props {
  notification: INotification
}

const Notification: React.FC<Props> = ({ notification }) => {
  const unread = !notification.is_viewed

  const typeMark = useMemo(() => {
    switch (notification.type) {
      case 'new_post_reply': return 'FORUM'
      case 'new_thread_post': return 'FORUM'
      case 'forum_mention': return 'MENTION'
      case 'new_article_revision': return 'REVISION'
      case 'welcome': return 'WELCOME'
      case 'direct_message': return 'PM'
      case 'post_like': return 'LIKE'
      default: return 'INFO'
    }
  }, [notification.type])

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
                section: <a href={notification.section.url}>{notification.section.name}</a>,
              }}
            />{' '}
            &raquo;{' '}
            <a href={notification.category.url}>{notification.category.name}</a> &raquo;{' '}
            <a href={notification.thread.url}>{notification.thread.name}</a>
          </Styled.PostFrom>
          <Styled.PostName>
            <a href={notification.post.url}>{notification.post.name || t('notifications.item.view-post')}</a>
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
              <a href={`/${notification.article.pageId}`}>{pageName}</a>
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
      return <Styled.TypeName>{t('notifications.item.type-welcome')}</Styled.TypeName>
    } else if (notification.type === 'post_like') {
      return (
        <>
          <Styled.TypeName>{t('notifications.item.type-post-like')}</Styled.TypeName>
          <Styled.PostFrom>
            <Trans id="notifications.item.liked-by" children={{ author: <UserView data={notification.author} /> }} />{' '}
            &raquo; <a href={notification.thread.url}>{notification.thread.name}</a>
          </Styled.PostFrom>
          <Styled.PostName>
            <a href={notification.post.url}>{notification.post.name || t('notifications.item.view-post')}</a>
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
      <Styled.TypeMark unread={unread}>{typeMark}</Styled.TypeMark>
      <Styled.Body>{body}</Styled.Body>
      <Styled.NotificationDate>{formatDate(new Date(notification.created_at))}</Styled.NotificationDate>
    </Styled.Container>
  )
}

export default Notification
