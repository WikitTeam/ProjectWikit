import { t } from '~util/i18n'

const borderSize = 10
const resizeMillis = 400

let links: HTMLAnchorElement[] = []
let current = 0

function byId<T extends HTMLElement>(id: string) {
  return document.getElementById(id) as T | null
}

function pageSize(): [number, number] {
  const root = document.documentElement
  return [Math.max(root.scrollWidth, root.clientWidth), Math.max(root.scrollHeight, window.innerHeight)]
}

function place() {
  const overlay = byId('jquery-overlay')
  const box = byId('jquery-lightbox')
  if (!overlay || !box) return
  const [width, height] = pageSize()
  overlay.style.width = `${width}px`
  overlay.style.height = `${height}px`
  box.style.top = `${window.scrollY + window.innerHeight / 10}px`
  box.style.left = `${window.scrollX}px`
}

function close() {
  byId('jquery-lightbox')?.remove()
  byId('jquery-overlay')?.remove()
  document.removeEventListener('keydown', onKey)
  window.removeEventListener('resize', place)
  links = []
}

function go(index: number) {
  if (index < 0 || index >= links.length || index === current) return
  current = index
  show()
}

function onKey(e: KeyboardEvent) {
  const key = e.key.toLowerCase()
  if (key === 'escape' || key === 'c' || key === 'x') close()
  else if (key === 'arrowleft' || key === 'p') go(current - 1)
  else if (key === 'arrowright' || key === 'n') go(current + 1)
}

function fit(width: number, height: number): [number, number] {
  const maxWidth = document.documentElement.clientWidth - 4 * borderSize
  const maxHeight = window.innerHeight * 0.8 - 2 * borderSize
  const scale = Math.min(1, maxWidth / width, maxHeight / height)
  return [Math.max(1, Math.round(width * scale)), Math.max(1, Math.round(height * scale))]
}

function show() {
  const image = byId<HTMLImageElement>('lightbox-image')
  const frame = byId('lightbox-container-image-box')
  const nav = byId('lightbox-nav')
  const data = byId('lightbox-container-image-data-box')
  const number = byId('lightbox-image-details-currentNumber')
  const caption = byId('lightbox-image-details-caption')
  const prev = byId('lightbox-nav-btnPrev')
  const next = byId('lightbox-nav-btnNext')
  if (!image || !frame || !nav || !data || !number || !caption || !prev || !next) return

  const shown = current
  for (const one of [image, nav, data]) one.style.display = 'none'
  byId('lightbox-loading')!.style.display = ''

  const loader = new Image()
  const reveal = (naturalWidth: number, naturalHeight: number) => {
    if (shown !== current || !byId('jquery-lightbox')) return
    const [width, height] = fit(naturalWidth, naturalHeight)
    image.src = loader.src
    image.width = width
    image.style.height = 'auto'
    image.style.visibility = 'hidden'
    image.style.display = ''
    const drawnWidth = image.offsetWidth || width
    const drawnHeight = image.offsetHeight || height
    image.style.display = 'none'
    image.style.visibility = ''

    const before = [frame.offsetWidth, frame.offsetHeight]
    frame.style.width = `${drawnWidth + 2 * borderSize}px`
    frame.style.height = `${drawnHeight + 2 * borderSize}px`
    data.style.width = `${drawnWidth}px`
    prev.style.height = next.style.height = `${drawnHeight + 2 * borderSize}px`
    const resized = before[0] !== drawnWidth + 2 * borderSize || before[1] !== drawnHeight + 2 * borderSize

    window.setTimeout(
      () => {
        if (shown !== current) return
        byId('lightbox-loading')!.style.display = 'none'
        image.style.display = ''
        const title = links[current].getAttribute('title')
        caption.textContent = title ?? ''
        caption.style.display = title ? '' : 'none'
        number.textContent = t('articles.lightbox.number', { current: current + 1, total: links.length })
        number.style.display = links.length > 1 ? '' : 'none'
        data.style.display = ''
        nav.style.display = ''
        prev.style.visibility = current > 0 ? '' : 'hidden'
        next.style.visibility = current < links.length - 1 ? '' : 'hidden'
        for (const near of [current - 1, current + 1]) {
          if (near >= 0 && near < links.length) new Image().src = links[near].href
        }
      },
      resized ? resizeMillis : 0,
    )
  }
  loader.onload = () => reveal(loader.naturalWidth || loader.width, loader.naturalHeight || loader.height)
  loader.onerror = () => reveal(250, 250)
  loader.src = links[current].href
}

function build() {
  const overlay = document.createElement('div')
  overlay.id = 'jquery-overlay'
  overlay.style.backgroundColor = '#000'
  overlay.style.opacity = '0.8'

  const box = document.createElement('div')
  box.id = 'jquery-lightbox'
  box.innerHTML =
    '<div id="lightbox-container-image-box"><div id="lightbox-container-image"><img id="lightbox-image" alt="">' +
    '<div id="lightbox-nav"><a href="#" id="lightbox-nav-btnPrev"></a><a href="#" id="lightbox-nav-btnNext"></a></div>' +
    '<div id="lightbox-loading"><a href="#" id="lightbox-loading-link"></a></div></div></div>' +
    '<div id="lightbox-container-image-data-box"><div id="lightbox-container-image-data"><div id="lightbox-image-details">' +
    '<span id="lightbox-image-details-caption"></span><span id="lightbox-image-details-currentNumber"></span></div>' +
    '<div id="lightbox-secNav"><a href="#" id="lightbox-secNav-btnClose"></a></div></div></div>'

  document.body.append(overlay, box)
  const frame = byId('lightbox-container-image-box')!
  frame.style.transition = `width ${resizeMillis}ms, height ${resizeMillis}ms`
  byId('lightbox-secNav-btnClose')!.setAttribute('aria-label', t('articles.lightbox.close'))
  byId('lightbox-nav-btnPrev')!.setAttribute('aria-label', t('articles.lightbox.prev'))
  byId('lightbox-nav-btnNext')!.setAttribute('aria-label', t('articles.lightbox.next'))

  overlay.addEventListener('click', close)
  box.addEventListener('click', e => {
    const target = (e.target as HTMLElement).closest('a')
    e.preventDefault()
    if (target?.id === 'lightbox-nav-btnPrev') go(current - 1)
    else if (target?.id === 'lightbox-nav-btnNext') go(current + 1)
    else close()
  })
}

function open(link: HTMLAnchorElement) {
  close()
  const gallery = link.closest('.gallery-box')
  links = gallery ? Array.from(gallery.querySelectorAll<HTMLAnchorElement>('a.with-lb')).filter(a => a.href) : [link]
  current = Math.max(links.indexOf(link), 0)

  build()
  place()
  document.addEventListener('keydown', onKey)
  window.addEventListener('resize', place)
  show()
}

export function makeLightbox() {
  document.addEventListener('click', e => {
    const link = (e.target as HTMLElement)?.closest?.('.gallery-box a.with-lb') as HTMLAnchorElement | null
    if (!link || !link.href) return
    e.preventDefault()
    open(link)
  })
}
