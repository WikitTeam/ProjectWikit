import { t } from '~util/i18n'
import * as React from 'react'
import { useEffect, useState } from 'react'
import { useTheme } from 'styled-components'
import { ProfilePage } from '~reactive/containers/page'
import { getOwnLikedPosts, LikedPostListing } from '~api/own-lists'
import useConstCallback from '~util/const-callback'
import Loader from '~util/loader'
import Pager from '~reactive/pages/own-lists/Pager'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'

const LikedPosts: React.FC = () => {
  const theme = useTheme()
  const [listing, setListing] = useState<LikedPostListing>()
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let live = true
    setLoading(true)
    getOwnLikedPosts(page)
      .then(resp => {
        if (live) setListing(resp)
      })
      .catch(e => console.error('Failed to fetch liked posts', e))
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [page])

  const onPage = useConstCallback((to: number) => setPage(to))

  return (
    <ProfilePage crumb={t('own-likes.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('own-likes.breadcrumb-profile')}</b><span className="sep">/</span>{t('own-likes.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{t('own-likes.title')}</Styled.H1>
      </Styled.SectionHead>
      {listing && <Styled.Count>{t('own-likes.count', { count: listing.total })}</Styled.Count>}
      {loading && (
        <Styled.LoaderContainer>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!loading && listing && listing.total === 0 && <Styled.EmptyMessage>{t('own-likes.empty')}</Styled.EmptyMessage>}
      {!loading && listing && listing.total > 0 && (
        <>
          <Styled.List>
            {listing.posts.map(one => (
              <Styled.Item key={one.postId}>
                <Styled.Title href={one.url}>{one.name || t('own-likes.untitled')}</Styled.Title>
                <Styled.Name>{one.threadName}</Styled.Name>
                <Styled.Added>{one.likedAt.slice(0, 10)}</Styled.Added>
              </Styled.Item>
            ))}
          </Styled.List>
          <Pager page={listing.page} pages={listing.pages} onPage={onPage} />
        </>
      )}
    </ProfilePage>
  )
}

export default LikedPosts
