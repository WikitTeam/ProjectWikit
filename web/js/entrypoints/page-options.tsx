import { t } from '~util/i18n'
import React, { useEffect, useState } from 'react'
import ReactDOM from 'react-dom'
import { sprintf } from 'sprintf-js'
import { NotificationSubscriptionData, subscribeToNotifications, unsubscribeFromNotifications } from '../api/notifications'
import { RatingMode } from '../api/rate'
import ArticleAuthorship from '../articles/article-authorship'
import ArticleBacklinksView from '../articles/article-backlinks'
import ArticleChild from '../articles/article-child'
import ArticleDelete from '../articles/article-delete'
import ArticleEditor from '../articles/article-editor'
import ArticleFiles from '../articles/article-files'
import ArticleHistory from '../articles/article-history'
import ArticleLock from '../articles/article-lock'
import ArticleParent from '../articles/article-parent'
import ArticleRating from '../articles/article-rating'
import ArticleRename from '../articles/article-rename'
import ArticleSource from '../articles/article-source'
import ArticleTags from '../articles/article-tags'
import useConstCallback from '../util/const-callback'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  pageId: string
  optionsEnabled?: boolean
  editable?: boolean
  lockable?: boolean
  tagable?: boolean
  rating?: number
  ratingVotes?: number
  ratingMode?: RatingMode
  pathParams?: { [key: string]: string }
  canRate?: boolean
  canDelete?: boolean
  canComment?: boolean
  canViewComments?: boolean
  commentThread?: string
  commentCount?: number
  canCreateTags?: boolean
  canManageFiles?: boolean
  canRename?: boolean
  canCreateHere?: boolean
  canManageAuthors?: boolean
  canResetVotes?: boolean
  canWatch?: boolean
  isWatching?: boolean
  preferences?: { [key: string]: any }
}

type SubViewType =
  | 'edit'
  | 'rating'
  | 'tags'
  | 'history'
  | 'source'
  | 'parent'
  | 'child'
  | 'lock'
  | 'rename'
  | 'files'
  | 'delete'
  | 'backlinks'
  | 'authorship'
  | null

const PageOptions: React.FC<Props> = ({
  pageId,
  optionsEnabled,
  editable,
  lockable,
  tagable,
  rating,
  ratingVotes,
  ratingMode,
  pathParams,
  canRate,
  canDelete,
  canComment,
  canViewComments,
  commentThread,
  commentCount,
  canCreateTags,
  canManageFiles,
  canRename,
  canCreateHere,
  canManageAuthors,
  canResetVotes,
  canWatch,
  isWatching,
  preferences,
}: Props) => {
  const [subView, setSubView] = useState<SubViewType>(null)
  const [extOptions, setExtOptions] = useState(false)
  const [isNewEditor, setIsNewEditor] = useState(false)
  const [isNowWatching, setIsNowWatching] = useState(isWatching)
  const [isSaving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [onCancelNewEditor, setOnCancelNewEditor] = useState<() => void>()

  useEffect(() => {
    ;(window as any)._openNewEditor = (func?: () => void) => {
      setTimeout(() => window.scrollTo(0, document.body.scrollHeight), 0)
      setOnCancelNewEditor(func)
      if (optionsEnabled) {
        setSubView('edit')
      } else {
        setIsNewEditor(true)
      }
    }
    if (pathParams?.['edit']) (window as any)._openNewEditor()
  }, [])

  const onEdit = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('edit')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    }, 1)
  })

  const onRate = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('rating')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    }, 1)
  })

  const onTags = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('tags')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    }, 1)
  })

  const onCancelSubView = useConstCallback(() => {
    if (isNewEditor && onCancelNewEditor) {
      onCancelNewEditor()
    }
    setSubView(null)
    setIsNewEditor(false)
  })

  const onHistory = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('history')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onFiles = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('files')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onSource = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('source')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onParent = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('parent')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onChild = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('child')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onLock = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('lock')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onRename = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('rename')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onDelete = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('delete')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onBacklinks = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('backlinks')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onAuthorship = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setSubView('authorship')
    setTimeout(() => {
      window.scrollTo(window.scrollX, document.body.scrollHeight)
    })
  })

  const onWatch = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()

    let request: NotificationSubscriptionData = {}

    if (e.target.id === 'watchPage') request = { pageId }
    else if (e.target.id === 'watchThread') request = { forumThreadId: +(pathParams?.t ?? '-1') }

    const action = isNowWatching ? unsubscribeFromNotifications : subscribeToNotifications

    let loadDelay = setTimeout(() => {
      setSaving(true)
    }, 1000)
    action(request)
      .then(() => {
        setIsNowWatching(!isNowWatching)
      })
      .catch(e => {
        setError(e.error || t('common.server-unreachable'))
      })
      .finally(() => {
        clearTimeout(loadDelay)
        setSaving(false)
      })
  })

  const onCloseError = useConstCallback(() => {
    setError('')
  })

  const toggleExtOptions = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setExtOptions(!extOptions)
  })

  const renderRating = useConstCallback(() => {
    if (ratingMode === 'updown') {
      return sprintf('%+d', rating)
    } else if (ratingMode === 'stars') {
      return ratingVotes ? sprintf('%.1f', rating) : '—'
    } else {
      return 'n/a'
    }
  })

  const renderSubView = useConstCallback(() => {
    return ReactDOM.createPortal(pickSubView(), document.getElementById('action-area')!)
  })

  const pickSubView = useConstCallback(() => {
    switch (subView) {
      case 'edit':
        return (
          <ArticleEditor
            pageId={pageId}
            pathParams={pathParams}
            useAdvancedEditor={preferences?.['qol__advanced_source_editor_enabled'] === true}
            onClose={onCancelSubView}
            previewTitleElement={document.getElementById('page-title') ?? undefined}
            previewBodyElement={document.getElementById('page-content') ?? undefined}
            previewStyleElement={document.getElementById('computed-style') ?? undefined}
          />
        )

      case 'rating':
        return (
          <ArticleRating
            pageId={pageId}
            rating={rating ?? 0}
            canEdit={Boolean(editable)}
            canResetVotes={Boolean(canResetVotes)}
            onClose={onCancelSubView}
          />
        )

      case 'tags':
        return <ArticleTags pageId={pageId} onClose={onCancelSubView} canCreateTags={canCreateTags} />

      case 'history':
        return <ArticleHistory pageId={pageId} pathParams={pathParams} onClose={onCancelSubView} />

      case 'source':
        return <ArticleSource pageId={pageId} onClose={onCancelSubView} />

      case 'parent':
        return <ArticleParent pageId={pageId} onClose={onCancelSubView} />

      case 'child':
        return <ArticleChild pageId={pageId} onClose={onCancelSubView} />

      case 'lock':
        return <ArticleLock pageId={pageId} onClose={onCancelSubView} />

      case 'rename':
        return <ArticleRename pageId={pageId} onClose={onCancelSubView} />

      case 'files':
        return <ArticleFiles pageId={pageId} onClose={onCancelSubView} editable={Boolean(canManageFiles)} />

      case 'delete':
        return <ArticleDelete pageId={pageId} canDelete={canDelete} canRename={canRename} onClose={onCancelSubView} />

      case 'backlinks':
        return <ArticleBacklinksView pageId={pageId} onClose={onCancelSubView} />

      case 'authorship':
        return <ArticleAuthorship user={null} pageId={pageId} onClose={onCancelSubView} editable={canManageAuthors} />

      default:
        return null
    }
  })

  if (isNewEditor) {
    return (
      <ArticleEditor
        pageId={pageId}
        isNew
        pathParams={pathParams}
        useAdvancedEditor={preferences?.['qol__advanced_source_editor_enabled'] === true}
        onClose={onCancelSubView}
        previewTitleElement={document.getElementById('page-title') ?? undefined}
        previewBodyElement={document.getElementById('page-content') ?? undefined}
      />
    )
  }

  if (!optionsEnabled) {
    return null
  }

  return (
    <>
      {canWatch && (
        <div className="page-watch-options">
          {pageId === 'forum:thread' && (
            <a href="#" id="watchThread" onClick={onWatch}>
              {isNowWatching ? t('page-options.watch-stop') : t('page-options.watch-start')}
            </a>
          )}
          {!pageId.startsWith('forum:') && (
            <a href="#" id="watchPage" onClick={onWatch}>
              {isNowWatching ? t('page-options.watch-stop') : t('page-options.watch-start')}
            </a>
          )}
        </div>
      )}
      {isSaving && (
        <WikidotModal isLoading>
          <p>{t('page-options.saving')}</p>
        </WikidotModal>
      )}
      {error && (
        <WikidotModal buttons={[{ title: t('page-options.error-dismiss'), onClick: onCloseError }]} isError>
          <p>
            <strong>{t('page-options.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <div id="page-options-bottom" className="page-options-bottom">
        {editable && (
          <a id="edit-button" className="btn btn-default" href="#" onClick={onEdit}>
            {t('page-options.edit')}
          </a>
        )}
        {ratingMode != 'disabled' && (
          <a id="pagerate-button" className="btn btn-default" href="#" onClick={onRate}>
            {t('page-options.rate')} ({renderRating()})
          </a>
        )}
        {tagable && (
          <a id="tags-button" className="btn btn-default" href="#" onClick={onTags}>
            {t('page-options.tags')}
          </a>
        )}
        {canViewComments && (
          <a id="discuss-button" className="btn btn-default" href={commentThread || '/forum/start'}>
            {t('page-options.discuss')} ({commentCount || 0})
          </a>
        )}
        <a id="history-button" className="btn btn-default" href="#" onClick={onHistory}>
          {t('page-options.history')}
        </a>
        <a id="files-button" className="btn btn-default" href="#" onClick={onFiles}>
          {t('page-options.files')}
        </a>
        <a id="more-options-button" className="btn btn-default" href="#" onClick={toggleExtOptions}>
          {extOptions ? '-' : '+'} {t('page-options.more')}
        </a>
      </div>
      {extOptions && (
        <div id="page-options-bottom-2" className="page-options-bottom form-actions">
          <a id="backlinks-button" className="btn btn-default" href="#" onClick={onBacklinks}>
            {t('page-options.backlinks')}
          </a>
          <a id="view-source-button" className="btn btn-default" href="#" onClick={onSource}>
            {t('page-options.source')}
          </a>
          <a id="view-authorship-button" className="btn btn-default" href="#" onClick={onAuthorship}>
            {t('page-options.authorship')}
          </a>
          {editable && (
            <a id="parent-page-button" className="btn btn-default" href="#" onClick={onParent}>
              {t('page-options.parent')}
            </a>
          )}
          {canCreateHere && (
            <a id="child-page-button" className="btn btn-default" href="#" onClick={onChild}>
              {t('page-options.new-child')}
            </a>
          )}
          {lockable && (
            <a id="page-block-button" className="btn btn-default" href="#" onClick={onLock}>
              {t('page-options.lock')}
            </a>
          )}
          {canRename && (
            <a id="rename-move-button" className="btn btn-default" href="#" onClick={onRename}>
              {t('page-options.rename')}
            </a>
          )}
          {(canRename || canDelete) && (
            <a id="delete-button" className="btn btn-default" href="#" onClick={onDelete}>
              {t('page-options.delete')}
            </a>
          )}
        </div>
      )}
      {renderSubView()}
    </>
  )
}

export default PageOptions
