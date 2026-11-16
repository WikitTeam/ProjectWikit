import * as React from 'react'
import { t } from '~util/i18n'
import Loader from '~util/loader'
import { renderTo, unmountFromRoot } from '~util/react-render-into'
import RatingTrend from '../articles/rating-trend'

interface DayItem {
  kind: 'edit' | 'vote' | 'post'
  title: string
  url: string
  site: string
  at: string
  note?: string
}

const perPage = 10

function el<K extends keyof HTMLElementTagNameMap>(tag: K, className?: string, text?: string): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag)
  if (className) node.className = className
  if (text !== undefined) node.textContent = text
  return node
}

function attachScoreTrend() {
  const mount = document.querySelector<HTMLElement>('.score-trend-mount[data-points]')
  if (!mount) return
  let points: Array<[number, number]>
  try {
    points = JSON.parse(mount.dataset.points!)
  } catch {
    return
  }
  const caption = mount.dataset.caption || ''
  mount.innerHTML = ''
  renderTo(mount, <RatingTrend points={points} stars={false} caption={() => caption} wide />)
}

function attachPinPicker() {
  const picker = document.querySelector<HTMLDetailsElement>('details.pin-picker')
  const form = picker?.querySelector<HTMLFormElement>('.pin-picker-panel')
  if (!picker || !form) return
  const limit = Number(form.dataset.limit) || 0
  const boxes = [...form.querySelectorAll<HTMLInputElement>('input[name="article_id"]')]

  const cap = () => {
    const full = limit > 0 && boxes.filter(b => b.checked).length >= limit
    for (const box of boxes) box.disabled = full && !box.checked
  }
  cap()
  form.addEventListener('change', cap)

  form.querySelector<HTMLInputElement>('.pin-picker-filter')?.addEventListener('input', e => {
    const wanted = (e.target as HTMLInputElement).value.trim().toLowerCase()
    for (const box of boxes) {
      const row = box.closest('li')!
      row.hidden = wanted !== '' && !row.textContent!.toLowerCase().includes(wanted)
    }
  })

  document.addEventListener('click', e => {
    if (picker.open && !picker.contains(e.target as Node)) picker.open = false
  })
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && picker.open) picker.open = false
  })
}

export function attachProfileActivity() {
  attachScoreTrend()
  attachPinPicker()
  const heatmap = document.querySelector<HTMLElement>('.heatmap[data-days]')
  const panel = document.querySelector<HTMLElement>('.activity-day')
  if (!heatmap || !panel) return

  let picked: HTMLElement | null = null
  let items: DayItem[] = []
  let page = 0
  let request = 0
  let loading: HTMLElement | null = null

  const show = (...parts: HTMLElement[]) => {
    if (loading) unmountFromRoot(loading)
    loading = null
    panel.replaceChildren(...parts)
  }

  const close = () => {
    picked?.classList.remove('picked')
    picked = null
    panel.hidden = true
    show()
  }

  const draw = (day: string) => {
    const head = el('div', 'activity-day-head')
    head.append(el('strong', undefined, t('profile-day.title', { date: day, count: items.length })))
    const shut = el('button', 'activity-day-close', t('profile-day.close'))
    shut.type = 'button'
    shut.addEventListener('click', close)
    head.append(shut)

    const list = el('ol')
    for (const item of items.slice(page * perPage, (page + 1) * perPage)) {
      const row = el('li')
      row.append(el('span', 'kind', t(`profile-day.kind-${item.kind}`)))
      const link = el('a', 'title', item.title)
      link.href = item.url
      link.title = item.site ? `${item.title} (${item.site})` : item.title
      row.append(link)
      row.append(el('span', 'note', item.note || ''))
      row.append(el('time', undefined, item.at))
      list.append(row)
    }
    const parts: HTMLElement[] = [head, list]
    if (!items.length) parts.push(el('div', 'empty', t('profile-day.empty')))

    const pages = Math.ceil(items.length / perPage)
    if (pages > 1) {
      const pager = el('div', 'activity-day-pager')
      const prev = el('button', undefined, t('profile-day.prev'))
      const next = el('button', undefined, t('profile-day.next'))
      prev.type = next.type = 'button'
      prev.disabled = page === 0
      next.disabled = page >= pages - 1
      prev.addEventListener('click', () => {
        page--
        draw(day)
      })
      next.addEventListener('click', () => {
        page++
        draw(day)
      })
      pager.append(prev, el('span', undefined, `${page + 1} / ${pages}`), next)
      parts.push(pager)
    }
    show(...parts)
  }

  const open = async (cell: HTMLElement) => {
    if (cell === picked) {
      close()
      return
    }
    picked?.classList.remove('picked')
    picked = cell
    cell.classList.add('picked')
    const day = cell.dataset.day!
    const mine = ++request
    panel.hidden = false
    show()
    loading = el('div', 'activity-day-loader')
    panel.append(loading)
    renderTo(loading, <Loader />)
    const query = new URLSearchParams({ day })
    const site = document.querySelector<HTMLSelectElement>('select[name="site"]')?.value
    if (site) query.set('site', site)
    try {
      const resp = await fetch(`${heatmap.dataset.days}?${query}`, { credentials: 'same-origin' })
      if (!resp.ok) throw new Error(String(resp.status))
      const body = (await resp.json()) as { items: DayItem[] }
      if (mine !== request) return
      items = body.items
      page = 0
      draw(day)
    } catch {
      if (mine !== request) return
      show(el('div', 'empty', t('profile-day.failed')))
    }
  }

  heatmap.addEventListener('click', e => {
    const cell = (e.target as HTMLElement).closest<HTMLElement>('i[data-day]')
    if (cell) open(cell)
  })
  heatmap.addEventListener('keydown', e => {
    const cell = (e.target as HTMLElement).closest<HTMLElement>('i[data-day]')
    if (cell && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault()
      open(cell)
    }
  })
}
