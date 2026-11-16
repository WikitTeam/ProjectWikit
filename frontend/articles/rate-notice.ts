import { t } from '~util/i18n'
import { fetchPageRating, ModuleRateResponse, ratePage } from '../api/rate'
import { showConfirmModal, showErrorModal, showInfoModal } from '../util/wikidot-modal'

type Format = (vote: number) => string

export const upDownVote: Format = vote => (vote > 0 ? `+${vote}` : String(vote))
export const starsVote: Format = vote => t('articles.rate-notice.stars', { count: vote })

const failed = (e: any) => showErrorModal(e?.error || t('common.server-unreachable'))

export async function castVote(pageId: string, value: number | null, format: Format, apply: (result: ModuleRateResponse) => void) {
  let current: number | null
  try {
    current = (await fetchPageRating(pageId)).vote ?? null
  } catch (e) {
    failed(e)
    return
  }

  const send = async (notice?: string) => {
    try {
      apply(await ratePage({ pageId, value }))
      if (notice) showInfoModal(notice)
    } catch (e) {
      failed(e)
    }
  }

  if (value === null) {
    if (current === null) {
      showInfoModal(t('articles.rate-notice.none'))
      return
    }
    showConfirmModal(t('articles.rate-notice.confirm-cancel', { vote: format(current) }), t('util.wikidot-modal.confirm'), () => send())
    return
  }
  if (current === null) {
    await send(t('articles.rate-notice.voted', { vote: format(value) }))
    return
  }
  if (current === value) {
    showInfoModal(t('articles.rate-notice.same', { vote: format(value) }))
    return
  }
  showConfirmModal(t('articles.rate-notice.confirm-change', { before: format(current), after: format(value) }), t('util.wikidot-modal.confirm'), () =>
    send(),
  )
}
