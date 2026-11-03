import * as React from 'react'
import { t } from '~util/i18n'
import Loader from '~util/loader'
import { renderTo, unmountFromRoot } from '~util/react-render-into'

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

export function attachProfileActivity() {
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
