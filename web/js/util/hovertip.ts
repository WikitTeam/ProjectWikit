// Wikidot themes style `.hovertip` and its `.content` child, so the markup has
// to keep that shape even though nothing in this file reads those rules.

export function hovertipContainer(): HTMLElement {
  let container = document.getElementById('odialog-hovertips')
  if (!container) {
    container = document.createElement('div')
    container.setAttribute('id', 'odialog-hovertips')
    Object.assign(container.style, {
      position: 'absolute',
      zIndex: '100',
      top: '0',
      width: '100%',
    })
    document.body.appendChild(container)
  }
  return container
}

// Site themes reach for .hovertip to restyle it, and one of them wins over a
// plain inline display. The tip is ours, so it declares its own visibility the
// only way nothing else can override.
export function show(tip: HTMLElement) {
  tip.style.setProperty('display', 'block', 'important')
}

export function hide(tip: HTMLElement) {
  tip.style.setProperty('display', 'none', 'important')
}

function position(tip: HTMLElement, x: number, y: number) {
  tip.style.left = '0'
  tip.style.top = '0'
  const r = tip.getBoundingClientRect()

  const centeredX = x - r.width / 2 + 4
  if (centeredX + r.width > window.innerWidth) {
    x = Math.max(0, window.innerWidth - r.width)
  } else if (centeredX < 0) {
    x = 0
  } else {
    x = centeredX
  }

  if (y + r.height + 8 > window.innerHeight && y - 8 > r.height) {
    y = Math.max(0, y - r.height - 8)
  } else {
    y += 8
  }

  tip.style.left = `${x}px`
  tip.style.top = `${y}px`
}

export function attachHovertip(node: HTMLElement, text: string | (() => string)) {
  if ((node as any)._hovertip) {
    return
  }
  ;(node as any)._hovertip = true

  const container = hovertipContainer()

  let tip: HTMLElement | null = container.querySelector('.hovertip.w-plain-hovertip')
  if (!tip) {
    tip = document.createElement('div')
    tip.setAttribute('class', 'hovertip w-plain-hovertip')
    Object.assign(tip.style, { position: 'fixed', display: 'none' })
    tip.innerHTML = '<div class="content"></div>'
    container.appendChild(tip)
  }
  const content: HTMLElement = tip.querySelector('.content')!

  node.style.cursor = 'help'

  node.addEventListener('mouseover', e => {
    content.textContent = typeof text === 'function' ? text() : text
    show(tip)
    position(tip, e.clientX, e.clientY)
  })

  node.addEventListener('mousemove', e => {
    position(tip, e.clientX, e.clientY)
  })

  node.addEventListener('mouseout', () => {
    hide(tip)
    content.textContent = ''
  })
}
