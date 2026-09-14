// A menu entry that only opens its submenu is written without a link, so its
// label arrives as a bare text node that the theme's `li > a` rules cannot
// reach. Wrapping it produces the same element the linked form would have.
export function makeTopBar() {
  for (const li of document.querySelectorAll('#top-bar li')) {
    const first = li.firstChild
    if (!first || first.nodeType !== Node.TEXT_NODE) continue

    const label = (first.textContent ?? '').trim()
    if (!label) continue

    const link = document.createElement('a')
    link.setAttribute('href', 'javascript:;')
    link.textContent = label
    li.replaceChild(link, first)
  }
}
