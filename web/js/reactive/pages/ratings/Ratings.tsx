import { t } from '~util/i18n'
import * as React from 'react'
import { useEffect, useState } from 'react'
import { useTheme } from 'styled-components'
import { ProfilePage } from '~reactive/containers/page'
import { getOwnRatings, RatingListing } from '~api/own-lists'
import useConstCallback from '~util/const-callback'
import Loader from '~util/loader'
import Pager from '~reactive/pages/own-lists/Pager'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'

function signed(rate: number): string {
  const rounded = Number.isInteger(rate) ? String(rate) : rate.toFixed(1)
  return rate > 0 ? `+${rounded}` : rounded
}

const Ratings: React.FC = () => {
  const theme = useTheme()
  const [listing, setListing] = useState<RatingListing>()
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let live = true
    setLoading(true)
    getOwnRatings(page)
      .then(resp => {
        if (live) setListing(resp)
      })
      .catch(e => console.error('Failed to fetch ratings', e))
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [page])

  const onPage = useConstCallback((to: number) => setPage(to))

  return (
    <ProfilePage crumb={t('own-ratings.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('own-ratings.breadcrumb-profile')}</b><span className="sep">/</span>{t('own-ratings.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{t('own-ratings.title')}</Styled.H1>
      </Styled.SectionHead>
      {listing && <Styled.Count>{t('own-ratings.count', { count: listing.total })}</Styled.Count>}
      {loading && (
        <Styled.LoaderContainer>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!loading && listing && listing.total === 0 && <Styled.EmptyMessage>{t('own-ratings.empty')}</Styled.EmptyMessage>}
      {!loading && listing && listing.total > 0 && (
        <>
          <Styled.List>
            {listing.ratings.map(one => (
              <Styled.Item key={one.pageId}>
                <Styled.Title href={`/${one.pageId}`}>{one.title}</Styled.Title>
                <Styled.Name>{one.pageId}</Styled.Name>
                <Styled.Badge positive={one.rate > 0}>{signed(one.rate)}</Styled.Badge>
                <Styled.Added>{one.votedAt ? one.votedAt.slice(0, 10) : ''}</Styled.Added>
              </Styled.Item>
            ))}
          </Styled.List>
          <Pager page={listing.page} pages={listing.pages} onPage={onPage} />
        </>
      )}
    </ProfilePage>
  )
}

export default Ratings
