import { t } from '~util/i18n'
import * as React from 'react'
import { useEffect, useState } from 'react'
import { useTheme } from 'styled-components'
import { ProfilePage } from '~reactive/containers/page'
import { FavouriteListing, getFavourites } from '~api/favourites'
import useConstCallback from '~util/const-callback'
import Loader from '~util/loader'
import Pager from '~reactive/pages/own-lists/Pager'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'

const Favourites: React.FC = () => {
  const theme = useTheme()
  const [listing, setListing] = useState<FavouriteListing>()
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let live = true
    setLoading(true)
    getFavourites(page)
      .then(resp => {
        if (live) setListing(resp)
      })
      .catch(e => console.error('Failed to fetch favourites', e))
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [page])

  const onPage = useConstCallback((to: number) => setPage(to))

  return (
    <ProfilePage crumb={t('favourites.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('favourites.breadcrumb-profile')}</b><span className="sep">/</span>{t('favourites.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{t('favourites.title')}</Styled.H1>
      </Styled.SectionHead>
      {listing && <Styled.Count>{t('favourites.count', { count: listing.total })}</Styled.Count>}
      {loading && (
        <Styled.LoaderContainer>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!loading && listing && listing.total === 0 && <Styled.EmptyMessage>{t('favourites.empty')}</Styled.EmptyMessage>}
      {!loading && listing && listing.total > 0 && (
        <>
          <Styled.List>
            {listing.favourites.map(one => (
              <Styled.Item key={one.pageId}>
                <Styled.Title href={`/${one.pageId}`}>{one.title}</Styled.Title>
                <Styled.Name>{one.pageId}</Styled.Name>
                <Styled.Added>{one.addedAt.slice(0, 10)}</Styled.Added>
              </Styled.Item>
            ))}
          </Styled.List>
          <Pager page={listing.page} pages={listing.pages} onPage={onPage} />
        </>
      )}
    </ProfilePage>
  )
}

export default Favourites
