import { t } from '~util/i18n'
import * as React from 'react'
import { useCallback, useState } from 'react'
import { fetchPostLikes, likePost, PostLikesResponse, unlikePost } from '../api/likes'
import { UserData } from '../api/user'
import UserView from '../util/user-view'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  postId: number
  liked: boolean
  count: number
}

const PostLikes: React.FC<Props> = ({ postId, liked: initialLiked, count: initialCount }) => {
  const [liked, setLiked] = useState(initialLiked)
  const [count, setCount] = useState(initialCount)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string>()
  const [listing, setListing] = useState<PostLikesResponse>()
  const [loading, setLoading] = useState(false)

  const onToggle = useCallback(
    async (e: React.MouseEvent) => {
      e.preventDefault()
      if (saving) return
      setSaving(true)
      try {
        const result = liked ? await unlikePost(postId) : await likePost(postId)
        setLiked(result.liked)
        setCount(result.count)
      } catch (err: any) {
        setError(err.error || t('forum.likes.failed'))
      } finally {
        setSaving(false)
      }
    },
    [liked, postId, saving],
  )

  const openPage = useCallback(
    async (page: number) => {
      setLoading(true)
      try {
        setListing(await fetchPostLikes(postId, page))
      } catch (err: any) {
        setError(err.error || t('forum.likes.failed'))
      } finally {
        setLoading(false)
      }
    },
    [postId],
  )

  const onOpen = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      if (count === 0) return
      openPage(1)
    },
    [count, openPage],
  )

  return (
    <>
      {error && (
        <WikidotModal buttons={[{ title: t('forum.likes.dismiss'), onClick: () => setError(undefined) }]} isError>
          <p>
            <strong>{t('forum.likes.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      {listing && (
        <WikidotModal buttons={[{ title: t('forum.likes.close'), onClick: () => setListing(undefined) }]}>
          <LikerList listing={listing} loading={loading} onPage={openPage} />
        </WikidotModal>
      )}
      <a href="#" className="like-toggle" title={t('forum.likes.toggle')} onClick={onToggle}>
        <i className={`${liked ? 'fas' : 'far'} fa-thumbs-up`} />
      </a>
      <a href="#" className="like-count" title={t('forum.likes.who')} onClick={onOpen}>
        {count}
      </a>
    </>
  )
}

const LikerList: React.FC<{ listing: PostLikesResponse; loading: boolean; onPage: (page: number) => void }> = ({
  listing,
  loading,
  onPage,
}) => (
  <div className="w-post-likers">
    <h2>{t('forum.likes.title', { count: listing.count })}</h2>
    <div className="w-post-likers-list">
      {listing.users.map((user: UserData, i: number) => (
        <div className="w-post-liker" key={i}>
          <UserView data={user} />
        </div>
      ))}
    </div>
    {listing.pages > 1 && <Pager listing={listing} loading={loading} onPage={onPage} />}
  </div>
)

const Pager: React.FC<{ listing: PostLikesResponse; loading: boolean; onPage: (page: number) => void }> = ({
  listing,
  loading,
  onPage,
}) => {
  const pages = pagesAround(listing.page, listing.pages)
  return (
    <div className="pager">
      <span className="pager-no">{t('forum.likes.page-of', { page: listing.page, total: listing.pages })}</span>
      {listing.page > 1 && (
        <span className="target">
          <a href="#" onClick={step(onPage, listing.page - 1, loading)}>
            {t('forum.likes.previous')}
          </a>
        </span>
      )}
      {pages.map((page, i) =>
        page === null ? (
          <span className="dots" key={i}>
            ...
          </span>
        ) : page === listing.page ? (
          <span className="target current" key={i}>
            {page}
          </span>
        ) : (
          <span className="target" key={i}>
            <a href="#" onClick={step(onPage, page, loading)}>
              {page}
            </a>
          </span>
        ),
      )}
      {listing.page < listing.pages && (
        <span className="target">
          <a href="#" onClick={step(onPage, listing.page + 1, loading)}>
            {t('forum.likes.next')}
          </a>
        </span>
      )}
    </div>
  )
}

function step(onPage: (page: number) => void, page: number, loading: boolean) {
  return (e: React.MouseEvent) => {
    e.preventDefault()
    if (!loading) onPage(page)
  }
}

// The first, the last and a window around the current one, so a post with two
// hundred pages of likes still shows a pager that fits.
function pagesAround(current: number, total: number): Array<number | null> {
  const wanted = new Set<number>([1, total])
  for (let p = current - 2; p <= current + 2; p++) {
    if (p >= 1 && p <= total) wanted.add(p)
  }
  const sorted = Array.from(wanted).sort((a, b) => a - b)
  const out: Array<number | null> = []
  sorted.forEach((page, i) => {
    if (i > 0 && page - sorted[i - 1] > 1) out.push(null)
    out.push(page)
  })
  return out
}

export default PostLikes
