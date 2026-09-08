const containerId = 'pwikit-lightbox'

function close() {
  document.getElementById(containerId)?.remove()
  document.removeEventListener('keydown', onKey)
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

function open(url: string) {
  close()

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
  image.src = url
  image.alt = ''
  Object.assign(image.style, { maxWidth: '92vw', maxHeight: '92vh', boxShadow: '0 0 24px rgba(0, 0, 0, 0.6)' })

  overlay.appendChild(image)
  overlay.addEventListener('click', close)
  document.body.appendChild(overlay)
  document.addEventListener('keydown', onKey)
}

// One listener on the document rather than one per image, because a gallery can
// arrive with a module that rendered after the page did.
export function makeLightbox() {
  document.addEventListener('click', e => {
    const link = (e.target as HTMLElement)?.closest?.('.gallery-box a') as HTMLAnchorElement | null
    if (!link || !link.href) return
    e.preventDefault()
    open(link.href)
  })
}
