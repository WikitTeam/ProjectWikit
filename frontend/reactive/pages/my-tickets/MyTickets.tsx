import { t } from '~util/i18n'
import * as React from 'react'
import { useEffect, useState } from 'react'
import { useTheme } from 'styled-components'
import { ProfilePage } from '~reactive/containers/page'
import { getOwnTickets, TicketListing } from '~api/own-lists'
import useConstCallback from '~util/const-callback'
import Loader from '~util/loader'
import Pager from '~reactive/pages/own-lists/Pager'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'

const MyTickets: React.FC = () => {
  const theme = useTheme()
  const [listing, setListing] = useState<TicketListing>()
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let live = true
    setLoading(true)
    getOwnTickets(page)
      .then(resp => {
        if (live) setListing(resp)
      })
      .catch(e => console.error('Failed to fetch own tickets', e))
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [page])

  const onPage = useConstCallback((to: number) => setPage(to))

  return (
    <ProfilePage crumb={t('own-tickets.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('own-tickets.breadcrumb-profile')}</b><span className="sep">/</span>{t('own-tickets.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{t('own-tickets.title')}</Styled.H1>
      </Styled.SectionHead>
      {listing && <Styled.Count>{t('own-tickets.count', { count: listing.total })}</Styled.Count>}
      {loading && (
        <Styled.LoaderContainer>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!loading && listing && listing.total === 0 && <Styled.EmptyMessage>{t('own-tickets.empty')}</Styled.EmptyMessage>}
      {!loading && listing && listing.total > 0 && (
        <>
          <Styled.List>
            {listing.tickets.map(one => (
              <Styled.Ticket key={`${one.kind}-${one.id}`}>
                <Styled.TicketHead>
                  <Styled.Badge positive={one.status !== 'pending'}>{t(`own-tickets.status-${one.status}`)}</Styled.Badge>
                  <Styled.TicketKind>{t(`own-tickets.kind-${one.kind}`)}</Styled.TicketKind>
                  <span>{one.subject || t('own-tickets.untitled')}</span>
                  <Styled.Name>
                    <a href={one.url}>{one.site}</a>
                  </Styled.Name>
                  <Styled.Added>{one.createdAt.slice(0, 10)}</Styled.Added>
                </Styled.TicketHead>
                {one.reply && <Styled.TicketReply>{one.reply}</Styled.TicketReply>}
              </Styled.Ticket>
            ))}
          </Styled.List>
          <Pager page={listing.page} pages={listing.pages} onPage={onPage} />
        </>
      )}
    </ProfilePage>
  )
}

export default MyTickets
