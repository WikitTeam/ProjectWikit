import { t } from '~util/i18n'
import * as React from 'react'
import { UserData } from '../api/user'
import formatDate from '../util/date-format'
import UserView from '../util/user-view'
import { ForumPostPreviewData } from './forum-post-editor'

interface Props {
  preview: ForumPostPreviewData
  user: UserData
  isThread?: boolean
}

const ForumPostPreview: React.FC<Props> = ({ preview, user, isThread }) => {
  const previewDate = new Date()
  return (
    <>
      {isThread && <h2>{t('forum.post-preview.preview-label')}</h2>}
      <div className="forum-thread-box">
        {isThread && (
          <div className="description-block well">
            {preview.description && <div className="head">{t('forum.post-preview.description-label')}</div>}
            {preview.description}
            <div className="statistics">
              {t('forum.post-preview.author-label')} <UserView data={user} avatarHover />
              <br />
              {t('forum.post-preview.date-label')} {formatDate(previewDate)}
            </div>
          </div>
        )}
        <div id="thread-container" className="thread-container">
          <div id="thread-container-posts">
            <div className="post-container">
              {!isThread && <h2>{t('forum.post-preview.reply-label')}</h2>}
              <div className="post">
                <div className="long">
                  <div className="head">
                    <div className="title">{preview.name}</div>
                    <div className="info">
                      <UserView data={user} avatarHover />{' '}
                      <span className="odate" style={{ display: 'inline' }}>
                        {formatDate(previewDate)}
                      </span>
                    </div>
                  </div>
                  <div className="content" dangerouslySetInnerHTML={{ __html: preview.content }} />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

export default ForumPostPreview