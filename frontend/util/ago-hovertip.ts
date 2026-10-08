import { formatDuration } from './date-format'
import { attachHovertip } from './hovertip'
import { t } from './i18n'

export function attachAgoHovertip(node: HTMLElement, date: Date) {
  attachHovertip(node, () => t('articles.history.date-ago', { duration: formatDuration(Date.now() - date.getTime()) }))
}
