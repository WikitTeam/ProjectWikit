import { t } from '~util/i18n'
import { fetchArticle, updateArticle } from '../api/articles'
import { callModule, ModuleRenderResponse } from '../api/modules'
import { showErrorModal } from '../util/wikidot-modal'

function currentPageId(): string | null {
  const node = document.getElementById('page-options-container') as HTMLElement | null
  if (!node || !node.dataset.config) return null
  try {
    return JSON.parse(node.dataset.config).pageId ?? null
  } catch {
    return null
  }
}

function applyOperations(tags: Array<string>, operations: string): Array<string> {
  const next = new Set(tags)
  for (const operation of operations.split(/\s+/).filter(Boolean)) {
    if (operation.startsWith('-')) {
      next.delete(operation.slice(1))
    } else {
      next.add(operation.replace(/^\+/, ''))
    }
  }
  return Array.from(next)
}

async function setTags(event: Event, operations: string) {
  event.preventDefault()
  const pageId = currentPageId()
  if (!pageId) return

  try {
    // The tags come back from the server rather than off the page, so a tag
    // block that did not render cannot turn one click into a wipe.
    const article = await fetchArticle(pageId)
    await updateArticle(pageId, { pageId, tags: applyOperations(article.tags ?? [], operations) })
  } catch (e) {
    showErrorModal(e.error || t('common.server-unreachable'))
    return
  }
  window.location.reload()
}

function edit(event: Event) {
  event.preventDefault()
  document.getElementById('edit-button')?.click()
}

function files(event: Event) {
  event.preventDefault()
  document.getElementById('files-button')?.click()
}

async function comments(event: Event) {
  event.preventDefault()
  const container = document.getElementById('thread-container')
  const options = document.getElementById('comments-options-hidden')
  if (!container) return

  const thread = Number(container.dataset.thread || '0')
  if (!thread) {
    document.getElementById('discuss-button')?.click()
    return
  }

  try {
    const response = await callModule<ModuleRenderResponse>({
      module: 'forumthread',
      pageId: currentPageId() ?? undefined,
      method: 'render',
      params: { t: thread },
    })
    container.innerHTML = response.result
  } catch (e) {
    showErrorModal(e.error || t('common.server-unreachable'))
    return
  }
  if (options) options.remove()
}

// The markup calls these by name from an inline onclick, because themes reach
// their buttons through [onclick*="..."] and a listener would leave no match.
export function makeButtons() {
  ;(window as any).pwikit = { ...((window as any).pwikit || {}), setTags, edit, files, comments }
}
