import { t } from '~util/i18n'
import { ModuleRateResponse } from '../api/rate'
import { showInfoModal } from '../util/wikidot-modal'

type Format = (vote: number) => string

export const upDownVote: Format = vote => (vote > 0 ? `+${vote}` : String(vote))
export const starsVote: Format = vote => t('articles.rate-notice.stars', { count: vote })

export function noticeVote(result: ModuleRateResponse, format: Format) {
  const before = result.previousVote ?? null
  const after = result.vote ?? null
  if (after === null) {
    showInfoModal(before === null ? t('articles.rate-notice.none') : t('articles.rate-notice.cancelled'))
  } else if (before === null) {
    showInfoModal(t('articles.rate-notice.voted', { vote: format(after) }))
  } else if (before === after) {
    showInfoModal(t('articles.rate-notice.same', { vote: format(after) }))
  } else {
    showInfoModal(t('articles.rate-notice.changed', { before: format(before), after: format(after) }))
  }
}
