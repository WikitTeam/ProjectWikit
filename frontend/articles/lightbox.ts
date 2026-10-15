const containerId = 'pwikit-lightbox'

let links: HTMLAnchorElement[] = []
let current = 0

function close() {
  document.getElementById(containerId)?.remove()
  document.removeEventListener('keydown', onKey)
  links = []
}

function step(by: number) {
  if (links.length < 2) return
  current = (current + by + links.length) % links.length
  const image = document.querySelector<HTMLImageElement>(`#${containerId} img`)
  if (image) image.src = links[current].href
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
  else if (e.key === 'ArrowLeft') step(-1)
  else if (e.key === 'ArrowRight') step(1)
}

function open(link: HTMLAnchorElement) {
  close()

  const box = link.closest('.gallery-box')
  links = box ? Array.from(box.querySelectorAll<HTMLAnchorElement>('a.with-lb')) : [link]
  current = Math.max(links.indexOf(link), 0)

  const overlay = document.createElement('div')
  overlay.id = containerId
  Object.assign(overlay.style, {
    position: 'fixed',
    inset: '0',
    zIndex: '10000',
    background: 'rgba(0, 0, 0, 0.8)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    cursor: 'zoom-out',
  })

  const image = document.createElement('img')
  image.src = link.href
  image.alt = ''
  Object.assign(image.style, { maxWidth: '92vw', maxHeight: '92vh', boxShadow: '0 0 24px rgba(0, 0, 0, 0.6)' })
  if (links.length > 1) {
    image.style.cursor = 'pointer'
    image.addEventListener('click', e => {
      e.stopPropagation()
      const rect = image.getBoundingClientRect()
      step(e.clientX - rect.left < rect.width / 2 ? -1 : 1)
    })
  }

  overlay.appendChild(image)
  overlay.addEventListener('click', close)
  document.body.appendChild(overlay)
  document.addEventListener('keydown', onKey)
}

// One listener on the document rather than one per image, because a gallery can
// arrive with a module that rendered after the page did.
export function makeLightbox() {
  document.addEventListener('click', e => {
    const link = (e.target as HTMLElement)?.closest?.('.gallery-box a.with-lb') as HTMLAnchorElement | null
    if (!link || !link.href) return
    e.preventDefault()
    open(link)
  })
}
