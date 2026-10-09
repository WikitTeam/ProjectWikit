import * as React from 'react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTheme } from 'styled-components'
import { getOwnTicket, TicketDetail } from '~api/own-lists'
import { ProfilePage } from '~reactive/containers/page'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'
import { Paths } from '~reactive/paths'
import formatDate from '~util/date-format'
import { t } from '~util/i18n'
import Loader from '~util/loader'
import * as Detail from './MyTicket.styles'

const when = (iso: string | null) => (iso ? formatDate(new Date(iso), '%Y-%m-%d %H:%M') : '')

const MyTicket: React.FC = () => {
  const theme = useTheme()
  const { kind = '', id = '' } = useParams()
  const [ticket, setTicket] = useState<TicketDetail>()
  const [missing, setMissing] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let live = true
    setLoading(true)
    setMissing(false)
    getOwnTicket(kind, id)
      .then(resp => {
        if (live) setTicket(resp)
      })
      .catch(() => {
        if (live) setMissing(true)
      })
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [kind, id])

  const isReport = ticket?.kind === 'report'

  return (
    <ProfilePage crumb={t('own-tickets.detail-crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('own-tickets.breadcrumb-profile')}</b>
          <span className="sep">/</span>
          <Detail.Back to={Paths.myTickets}>{t('own-tickets.breadcrumb')}</Detail.Back>
        </Styled.Kicker>
        <Styled.H1>{ticket ? ticket.subject || t('own-tickets.untitled') : t('own-tickets.detail-crumb')}</Styled.H1>
      </Styled.SectionHead>
      {loading && (
        <Styled.LoaderContainer>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!loading && missing && <Styled.EmptyMessage>{t('own-tickets.not-found')}</Styled.EmptyMessage>}
      {!loading && ticket && (
        <>
          <Detail.Facts>
            <dt>{t('own-tickets.field-status')}</dt>
            <dd>
              <Styled.Badge positive={ticket.status !== 'pending'}>{t(`own-tickets.status-${ticket.status}`)}</Styled.Badge>
            </dd>
            <dt>{t('own-tickets.field-kind')}</dt>
            <dd>{t(`own-tickets.kind-${ticket.kind}`)}</dd>
            <dt>{t('own-tickets.field-site')}</dt>
            <dd>
              <a href={ticket.url}>{ticket.site}</a>
            </dd>
            {isReport && (
              <>
                <dt>{t('own-tickets.field-reported')}</dt>
                <dd>{ticket.subject}</dd>
              </>
            )}
            {ticket.sourcePage && (
              <>
                <dt>{t('own-tickets.field-page')}</dt>
                <dd>
                  <a href={ticket.sourceUrl}>{ticket.sourcePage}</a>
                </dd>
              </>
            )}
            <dt>{t('own-tickets.field-submitted')}</dt>
            <dd>{when(ticket.createdAt)}</dd>
            {ticket.reviewedAt && (
              <>
                <dt>{t('own-tickets.field-reviewed')}</dt>
                <dd>{when(ticket.reviewedAt)}</dd>
              </>
            )}
          </Detail.Facts>
          <Detail.Section>
            <Detail.Heading>{isReport ? t('own-tickets.field-reason') : t('own-tickets.field-body')}</Detail.Heading>
            <Detail.Text>{ticket.body}</Detail.Text>
          </Detail.Section>
          {isReport && ticket.messages.length > 0 && (
            <Detail.Section>
              <Detail.Heading>{t('own-tickets.field-messages')}</Detail.Heading>
              <Detail.Messages>
                {ticket.messages.map((one, i) => (
                  <li key={i}>
                    <Detail.MessageMeta>
                      {one.sender}
                      {one.createdAt && <span>{when(one.createdAt)}</span>}
                    </Detail.MessageMeta>
                    <Detail.Text>{one.body}</Detail.Text>
                  </li>
                ))}
              </Detail.Messages>
            </Detail.Section>
          )}
          <Detail.Section>
            <Detail.Heading>{t('own-tickets.field-reply')}</Detail.Heading>
            {ticket.reply ? <Detail.Reply>{ticket.reply}</Detail.Reply> : <Styled.EmptyMessage>{t('own-tickets.no-reply')}</Styled.EmptyMessage>}
          </Detail.Section>
        </>
      )}
    </ProfilePage>
  )
}

export default MyTicket
