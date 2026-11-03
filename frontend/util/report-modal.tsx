import * as React from 'react'
import styled from 'styled-components'
import { t } from '~util/i18n'
import { wFetch } from './fetch-util'
import WikidotModal, { addUnmanagedModal, removeUnmanagedModal, showErrorModal, showInfoModal } from './wikidot-modal'

export type ReportTarget = { user_id: number } | { page: string } | { post_id: number }

const Reason = styled.textarea`
  display: block;
  width: 100%;
  min-height: 7em;
  margin: 8px 0 4px;
  box-sizing: border-box;
`

const Note = styled.p`
  margin: 0 0 4px;
  font-size: 0.92em;
  opacity: 0.8;
`

function siteTitle(): string {
  const meta = document.querySelector<HTMLMetaElement>('meta[property="og:site_name"]')?.content
  return meta || document.querySelector<HTMLElement>('[data-site-title]')?.dataset.siteTitle || document.location.host
}

export function showReportModal(target: ReportTarget, what: string) {
  let uuid: string | null = null
  let reason = ''
  let sending = false

  const close = () => {
    if (uuid) removeUnmanagedModal(uuid)
  }

  const submit = async () => {
    const trimmed = reason.trim()
    if (!trimmed || sending) return
    sending = true
    try {
      await wFetch('/pw-api/messages/report-content', { method: 'POST', sendJson: true, body: { ...target, reason: trimmed } })
      close()
      showInfoModal(t('report.sent'))
    } catch (err: any) {
      showErrorModal(err?.error || t('report.failed'))
    } finally {
      sending = false
    }
  }

  uuid = addUnmanagedModal(
    <WikidotModal
      buttons={[
        { title: t('report.cancel'), onClick: close },
        { title: t('report.submit'), onClick: submit, type: 'danger' },
      ]}
    >
      <p>
        <strong>{what}</strong>
      </p>
      <Note>{t('report.note', { site: siteTitle() })}</Note>
      <Reason
        placeholder={t('report.placeholder')}
        maxLength={2000}
        onChange={e => {
          reason = e.target.value
        }}
      />
    </WikidotModal>,
  )
}
