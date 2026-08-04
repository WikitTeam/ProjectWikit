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
      default: return 'INFO'
    }
  }, [notification.type])

  const body = useMemo(() => {
    if (notification.type === 'new_post_reply' || notification.type === 'new_thread_post' || notification.type === 'forum_mention') {
      const title = {
        new_post_reply: '回复了您的消息',
        new_thread_post: '新的论坛消息',
        forum_mention: '在论坛中提到您',
      }
      return (
        <>
          <Styled.TypeName>{title[notification.type as keyof typeof title]}</Styled.TypeName>
          <Styled.PostFrom>
            来自 <UserView data={notification.author} /> 在话题中 <a href={notification.section.url}>{notification.section.name}</a> &raquo;{' '}
            <a href={notification.category.url}>{notification.category.name}</a> &raquo;{' '}
            <a href={notification.thread.url}>{notification.thread.name}</a>
          </Styled.PostFrom>
          <Styled.PostName>
            <a href={notification.post.url}>{notification.post.name || '查看消息'}</a>
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
          <Styled.TypeName>关注的页面有新的编辑</Styled.TypeName>
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
              <Styled.RevisionCommentCaption>评论：</Styled.RevisionCommentCaption> {comment}
            </Styled.RevisionComment>
          )}
        </>
      )
    } else if (notification.type === 'welcome') {
      return <Styled.TypeName>欢迎来到本站</Styled.TypeName>
    } else if (notification.type === 'direct_message') {
      return (
        <>
          <Styled.TypeName>{notification.sender_name} 给你发了新私信</Styled.TypeName>
          <Styled.PostContent>{notification.preview}</Styled.PostContent>
          <Styled.PostName>
            <a href={`/-/messages/${notification.sender_id}`}>查看会话</a>
          </Styled.PostName>
        </>
      )
    } else {
      return <Styled.TypeName>通知渲染失败</Styled.TypeName>
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
