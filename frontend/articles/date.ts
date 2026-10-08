import formatDate, { formatDuration } from '../util/date-format'
import { attachHovertip } from '~util/hovertip'
import { attachAgoHovertip } from '~util/ago-hovertip'

export function makeDate(node: HTMLElement) {
  // hack: mark node as already processed because it was
  if ((node as any)._date) {
    return
  }
  ;(node as any)._date = true
  // end hack

  try {
    const defaultFormatHere = '%Y.%m.%d %H:%M'

    const timestamp = Number.parseInt(node.dataset.timestamp ?? '')

    // Everything after a pipe is a modifier, not part of the format. Feeding it
    // to the formatter would leave it in the page, since anything without a
    // percent passes through untouched.
    const [rawFormat, ...modifiers] = (node.dataset.format || defaultFormatHere).split('|')
    const format = rawFormat || defaultFormatHere

    const date = new Date(timestamp)

    let formatted
    try {
      formatted = formatDate(date, format)
    } catch (e) {
      formatted = formatDate(date, defaultFormatHere)
    }

    if (modifiers.includes('ago')) {
      const absolute = formatted
      attachHovertip(node, absolute)
      formatted = formatDuration(new Date().getTime() - date.getTime())
    } else if (modifiers.includes('agohover')) {
      attachHovertip(node, () => formatDuration(new Date().getTime() - date.getTime()))
    } else if (node.dataset.agoTip !== undefined) {
      attachAgoHovertip(node, date)
    }

    node.textContent = formatted
  } catch (e) {
    /* ... */
  }
}
