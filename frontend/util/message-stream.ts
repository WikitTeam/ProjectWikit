const STREAM_PATH = '/pw-api/messages/stream'
const FALLBACK_POLL_MS = 30000
const REOPEN_MS = 60000

type ChangeListener = () => void
type UnreadListener = (unread: number) => void

const changeListeners = new Set<ChangeListener>()
const unreadListeners = new Set<UnreadListener>()
let source: EventSource | null = null
let fallback: number | undefined
let reopen: number | undefined

function emitChange() {
  changeListeners.forEach(listener => listener())
}

function stopFallback() {
  window.clearInterval(fallback)
  window.clearTimeout(reopen)
  fallback = undefined
  reopen = undefined
}

function open() {
  if (source || typeof EventSource === 'undefined') {
    if (!source && fallback === undefined) fallback = window.setInterval(emitChange, FALLBACK_POLL_MS)
    return
  }
  const stream = new EventSource(STREAM_PATH)
  source = stream
  stream.addEventListener('messages', event => {
    stopFallback()
    try {
      const data = JSON.parse((event as MessageEvent).data)
      if (typeof data.unread === 'number') unreadListeners.forEach(listener => listener(data.unread))
    } catch {}
    emitChange()
  })
  stream.onerror = () => {
    if (stream.readyState !== EventSource.CLOSED) return
    stream.close()
    source = null
    if (changeListeners.size === 0) return
    if (fallback === undefined) fallback = window.setInterval(emitChange, FALLBACK_POLL_MS)
    reopen = window.setTimeout(() => {
      reopen = undefined
      if (changeListeners.size > 0) open()
    }, REOPEN_MS)
  }
}

function close() {
  stopFallback()
  source?.close()
  source = null
}

export function watchMessages(listener: ChangeListener): () => void {
  changeListeners.add(listener)
  open()
  return () => {
    changeListeners.delete(listener)
    if (changeListeners.size === 0) close()
  }
}

export function onUnreadMessages(listener: UnreadListener): () => void {
  unreadListeners.add(listener)
  return () => {
    unreadListeners.delete(listener)
  }
}
