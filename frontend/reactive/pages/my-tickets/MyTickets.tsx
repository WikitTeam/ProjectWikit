import * as React from 'react'
import { useEffect, useState } from 'react'
import { useTheme } from 'styled-components'
import { clearAllOwnTickets, clearOwnTickets, getOwnTickets, TicketEntry, TicketListing } from '~api/own-lists'
import { ProfilePage } from '~reactive/containers/page'
import * as Bar from '~reactive/pages/notifications/Notifications.styles'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'
import Pager from '~reactive/pages/own-lists/Pager'
import { Paths } from '~reactive/paths'
import useConstCallback from '~util/const-callback'
import { t } from '~util/i18n'
import Loader from '~util/loader'
import { showConfirmModal } from '~util/wikidot-modal'

const refOf = (one: TicketEntry) => `${one.kind}-${one.id}`

const MyTickets: React.FC = () => {
  const theme = useTheme()
  const [listing, setListing] = useState<TicketListing>()
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [clearing, setClearing] = useState(false)
  const [reload, setReload] = useState(0)

  useEffect(() => {
    let live = true
    setLoading(true)
    getOwnTickets(page)
      .then(resp => {
        if (live) {
          setListing(resp)
          setSelected(new Set())
        }
      })
      .catch(e => console.error('Failed to fetch own tickets', e))
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [page, reload])

  const onPage = useConstCallback((to: number) => setPage(to))

  const rows = listing?.tickets ?? []

  const toggle = useConstCallback((ref: string) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(ref)) next.delete(ref)
      else next.add(ref)
      return next
    })
  })

  const toggleAll = useConstCallback(() => {
    setSelected(prev => (prev.size === rows.length ? new Set<string>() : new Set(rows.map(refOf))))
  })

  const run = useConstCallback(async (action: () => Promise<unknown>) => {
    if (clearing) return
    setClearing(true)
    try {
      await action()
      setPage(1)
      setReload(n => n + 1)
    } catch (e: any) {
      console.error('Failed to clear own tickets', e)
    } finally {
      setClearing(false)
    }
  })

  const onClearSelected = useConstCallback(() => {
    if (selected.size === 0) return
    const items = rows.filter(one => selected.has(refOf(one))).map(one => ({ kind: one.kind, id: one.id }))
    showConfirmModal(t('own-tickets.confirm-clear-selected', { count: items.length }), t('own-tickets.confirm'), () => run(() => clearOwnTickets(items)))
  })

  const onClearAll = useConstCallback(() => {
    showConfirmModal(t('own-tickets.confirm-clear-all'), t('own-tickets.confirm'), () => run(() => clearAllOwnTickets()))
  })

  return (
    <ProfilePage crumb={t('own-tickets.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('own-tickets.breadcrumb-profile')}</b>
          <span className="sep">/</span>
          {t('own-tickets.breadcrumb')}
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
          <Bar.Toolbar>
            <Bar.ToolbarLabel>
              <input type="checkbox" checked={selected.size === rows.length && rows.length > 0} onChange={toggleAll} />
              {t('own-tickets.select-all')}
            </Bar.ToolbarLabel>
            <span>{t('own-tickets.selected-count', { count: selected.size })}</span>
            <Bar.ToolbarAction danger disabled={selected.size === 0 || clearing} onClick={onClearSelected}>
              {t('own-tickets.clear-selected')}
            </Bar.ToolbarAction>
            <Bar.ToolbarAction danger disabled={clearing} onClick={onClearAll}>
              {t('own-tickets.clear-all')}
            </Bar.ToolbarAction>
          </Bar.Toolbar>
          <Styled.List>
            {listing.tickets.map(one => (
              <Bar.SelectRow key={refOf(one)}>
                <input type="checkbox" checked={selected.has(refOf(one))} onChange={() => toggle(refOf(one))} />
                <Styled.Ticket>
                  <Styled.TicketHead>
                    <Styled.Badge positive={one.status !== 'pending'}>{t(`own-tickets.status-${one.status}`)}</Styled.Badge>
                    <Styled.TicketKind>{t(`own-tickets.kind-${one.kind}`)}</Styled.TicketKind>
                    <Styled.TicketTitle to={`${Paths.myTickets}/${one.kind}/${one.id}`}>
                      {one.subject || t('own-tickets.untitled')}
                    </Styled.TicketTitle>
                    <Styled.Name>
                      <a href={one.url}>{one.site}</a>
                    </Styled.Name>
                    <Styled.Added>{one.createdAt.slice(0, 10)}</Styled.Added>
                  </Styled.TicketHead>
                  {one.reply && <Styled.TicketReply>{one.reply}</Styled.TicketReply>}
                </Styled.Ticket>
              </Bar.SelectRow>
            ))}
          </Styled.List>
          <Pager page={listing.page} pages={listing.pages} onPage={onPage} />
        </>
      )}
    </ProfilePage>
  )
}

export default MyTickets
