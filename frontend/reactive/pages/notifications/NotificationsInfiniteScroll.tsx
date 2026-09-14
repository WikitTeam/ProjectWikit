import { t } from '~util/i18n'
import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { useTheme } from 'styled-components'
import { clearAllNotifications, clearNotifications, getNotifications, NotificationKind, Notification as INotification } from '~api/notifications'
import useConstCallback from '~util/const-callback'
import Loader from '~util/loader'
import Notification from './Notification'
import * as Styled from './Notifications.styles'

interface Props {
  batchSize: number
  showUnread: boolean
  kind: NotificationKind
  isForceUpdate?: () => boolean
}

const NotificationsInfiniteScroll: React.FC<Props> = ({ batchSize, showUnread, kind, isForceUpdate }) => {
  const theme = useTheme()
  const [items, setItems] = useState<INotification[]>([])
  const [cursor, setCursor] = useState(-1)
  const [hasMore, setHasMore] = useState(true)
  const [isFetching, setIsFetching] = useState(false)
  const loaderRef = useRef<HTMLDivElement | null>(null)
  const [isLoaderVisible, setIsLoaderVisible] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [clearing, setClearing] = useState(false)

  useEffect(() => {
    if (!isForceUpdate) return
    if (isForceUpdate()) {
      setIsFetching(false)
      setHasMore(true)
      setCursor(-1)
      setItems([])
    }
  })

  const handleVisibleChange = useConstCallback((newVisible: boolean) => {
    if (isLoaderVisible !== newVisible) {
      setIsLoaderVisible(newVisible)
    }
  })

  useEffect(() => {
    if (isLoaderVisible && !isFetching && hasMore) {
      loadMore()
    }
  }, [isFetching, isLoaderVisible])

  const loadMore = useConstCallback(async () => {
    while (isFetching) {
      return
    }

    setIsFetching(true)
    try {
      const resp = await getNotifications(cursor, batchSize, showUnread, true, kind)
      setItems(prev => [...prev, ...resp.notifications])
      setCursor(resp.cursor)

      if (resp.notifications.length < batchSize || resp.cursor === -1) {
        setHasMore(false)
      }
    } catch (e: any) {
      setHasMore(false)
      console.error('Failed to fetch more notifications', e)
    } finally {
      setIsFetching(false)
    }
  })

  useEffect(() => {
    const loader = loaderRef.current
    if (!loader) {
      return undefined
    }

    const observer = new IntersectionObserver(entries => {
      const newVisible = entries.some(x => x.isIntersecting && x.intersectionRatio > 0)
      handleVisibleChange(newVisible)
    })

    observer.observe(loader)

    return () => {
      observer.unobserve(loader)
    }
  }, [loaderRef.current])

  const toggle = useConstCallback((id: number) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  })

  const toggleAll = useConstCallback(() => {
    setSelected(prev => (prev.size === items.length ? new Set<number>() : new Set(items.map(x => x.id))))
  })

  const clearSelected = useConstCallback(async () => {
    if (selected.size === 0 || clearing) return
    setClearing(true)
    try {
      const ids = Array.from(selected)
      await clearNotifications(ids)
      setItems(prev => prev.filter(x => !selected.has(x.id)))
      setSelected(new Set())
    } catch (e: any) {
      console.error('Failed to clear notifications', e)
    } finally {
      setClearing(false)
    }
  })

  const clearEverything = useConstCallback(async () => {
    if (clearing) return
    setClearing(true)
    try {
      await clearAllNotifications(kind)
      setItems([])
      setSelected(new Set())
      setHasMore(false)
    } catch (e: any) {
      console.error('Failed to clear notifications', e)
    } finally {
      setClearing(false)
    }
  })

  return (
    <>
      {items.length > 0 && (
        <Styled.Toolbar>
          <Styled.ToolbarLabel>
            <input type="checkbox" checked={selected.size === items.length && items.length > 0} onChange={toggleAll} />
            {t('notifications.select-all')}
          </Styled.ToolbarLabel>
          <span>{t('notifications.selected-count', { count: selected.size })}</span>
          <Styled.ToolbarAction danger disabled={selected.size === 0 || clearing} onClick={clearSelected}>
            {t('notifications.clear-selected')}
          </Styled.ToolbarAction>
          <Styled.ToolbarAction danger disabled={clearing} onClick={clearEverything}>
            {t('notifications.clear-all')}
          </Styled.ToolbarAction>
        </Styled.Toolbar>
      )}
      <Styled.List>
        {items.map(item => (
          <Styled.SelectRow key={item.id}>
            <input type="checkbox" checked={selected.has(item.id)} onChange={() => toggle(item.id)} />
            <Notification notification={item} />
          </Styled.SelectRow>
        ))}
      </Styled.List>
      {hasMore && (
        <Styled.LoaderContainer ref={loaderRef}>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!hasMore && items.length === 0 && <Styled.EmptyMessage>{t('notifications.infinite-scroll.empty')}</Styled.EmptyMessage>}
    </>
  )
}

export default NotificationsInfiniteScroll
