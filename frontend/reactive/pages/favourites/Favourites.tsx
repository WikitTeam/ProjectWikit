import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { useTheme } from 'styled-components'
import { changeFavourites, FavouriteFolder, FavouriteListing, getFavourites } from '~api/favourites'
import { ProfilePage } from '~reactive/containers/page'
import * as Styled from '~reactive/pages/own-lists/OwnList.styles'
import Pager from '~reactive/pages/own-lists/Pager'
import useConstCallback from '~util/const-callback'
import { APIError } from '~util/fetch-util'
import { t } from '~util/i18n'
import Loader from '~util/loader'
import WikidotModal, { showConfirmModal } from '~util/wikidot-modal'
import * as F from './Favourites.styles'

const folderName = (f: FavouriteFolder) => (f.id === 0 ? t('favourites.default-folder') : f.name)

const Favourites: React.FC = () => {
  const theme = useTheme()
  const [listing, setListing] = useState<FavouriteListing>()
  const [folder, setFolder] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [reload, setReload] = useState(0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newPublic, setNewPublic] = useState(false)
  const [createError, setCreateError] = useState('')
  const creatingNow = useRef(false)

  useEffect(() => {
    let live = true
    setLoading(true)
    getFavourites(folder, page)
      .then(resp => {
        if (live) setListing(resp)
      })
      .catch(e => console.error('Failed to fetch favourites', e))
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [folder, page, reload])

  const run = async (change: Parameters<typeof changeFavourites>[0]) => {
    setBusy(true)
    setError('')
    try {
      await changeFavourites(change)
      setReload(n => n + 1)
      return true
    } catch (e) {
      setError(e instanceof APIError ? e.error : t('articles.favourite.failed'))
      return false
    } finally {
      setBusy(false)
    }
  }

  const onPage = useConstCallback((to: number) => setPage(to))

  const open = (id: number) => {
    setFolder(id)
    setPage(1)
    setError('')
  }

  const closeCreate = () => {
    setCreating(false)
    setNewName('')
    setNewPublic(false)
    setCreateError('')
  }

  const onCreate = async () => {
    if (newName.trim() === '') {
      setCreateError(t('favourites.name-required'))
      return
    }
    if (creatingNow.current) return
    creatingNow.current = true
    try {
      await changeFavourites({ action: 'create', name: newName, public: newPublic })
      closeCreate()
      setReload(n => n + 1)
    } catch (e) {
      setCreateError(e instanceof APIError ? e.error : t('articles.favourite.failed'))
    } finally {
      creatingNow.current = false
    }
  }

  const onVisibility = () => {
    if (!current) return
    const name = folderName(current)
    const making = !current.public
    showConfirmModal(
      making ? t('favourites.make-public-confirm', { name }) : t('favourites.make-private-confirm', { name }),
      making ? t('favourites.make-public') : t('favourites.make-private'),
      () => run({ action: 'visibility', folder, public: making }),
    )
  }

  const onDelete = () => {
    showConfirmModal(t('favourites.delete-confirm', { name: current?.name ?? '' }), t('favourites.delete'), async () => {
      if (await run({ action: 'delete', folder })) open(0)
    })
  }

  const folders = listing?.folders ?? []
  const current = folders.find(f => f.id === folder)

  return (
    <ProfilePage crumb={t('favourites.crumb')}>
      <Styled.SectionHead>
        <Styled.Kicker>
          <b>{t('favourites.breadcrumb-profile')}</b>
          <span className="sep">/</span>
          {t('favourites.breadcrumb')}
        </Styled.Kicker>
        <Styled.H1>{t('favourites.title')}</Styled.H1>
      </Styled.SectionHead>

      {folders.length > 0 && (
        <F.Folders>
          {folders.map(f => (
            <F.Folder key={f.id} current={f.id === folder} onClick={() => open(f.id)}>
              <span>{folderName(f)}</span>
              <span className="count">{f.count}</span>
              <F.Visibility>{f.public ? t('favourites.public') : t('favourites.private')}</F.Visibility>
            </F.Folder>
          ))}
          <F.Action onClick={() => setCreating(true)}>
            <i className="fas fa-plus" /> {t('favourites.create')}
          </F.Action>
        </F.Folders>
      )}

      {current && (
        <F.Toolbar>
          <span>{current.public ? t('favourites.public-hint') : t('favourites.private-hint')}</span>
          <F.Action disabled={busy} onClick={onVisibility}>
            {current.public ? t('favourites.make-private') : t('favourites.make-public')}
          </F.Action>
          {current.id !== 0 && (
            <F.Action danger disabled={busy} onClick={onDelete}>
              {t('favourites.delete')}
            </F.Action>
          )}
        </F.Toolbar>
      )}

      {error && <F.Error>{error}</F.Error>}

      {creating && (
        <WikidotModal
          buttons={[
            { title: t('favourites.cancel'), onClick: closeCreate },
            { title: t('favourites.create'), onClick: onCreate, type: 'primary' },
          ]}
        >
          <F.Dialog onSubmit={e => (e.preventDefault(), onCreate())}>
            <h1>{t('favourites.create')}</h1>
            <label className="field">
              {t('favourites.new-name')}
              <F.NameInput autoFocus value={newName} maxLength={64} onChange={e => setNewName(e.target.value)} />
            </label>
            <F.Check>
              <input type="radio" name="visibility" checked={!newPublic} onChange={() => setNewPublic(false)} />
              {t('favourites.private')}
              <F.Visibility>{t('favourites.private-hint')}</F.Visibility>
            </F.Check>
            <F.Check>
              <input type="radio" name="visibility" checked={newPublic} onChange={() => setNewPublic(true)} />
              {t('favourites.public')}
              <F.Visibility>{t('favourites.public-hint')}</F.Visibility>
            </F.Check>
            {createError && <F.Error>{createError}</F.Error>}
          </F.Dialog>
        </WikidotModal>
      )}

      {listing && <Styled.Count>{t('favourites.count', { count: listing.total })}</Styled.Count>}
      {loading && (
        <Styled.LoaderContainer>
          <Loader color={theme.primary} />
        </Styled.LoaderContainer>
      )}
      {!loading && listing && listing.total === 0 && (
        <Styled.EmptyMessage>{folder === 0 ? t('favourites.empty') : t('favourites.empty-folder')}</Styled.EmptyMessage>
      )}
      {!loading && listing && listing.total > 0 && (
        <>
          <Styled.List>
            {listing.favourites.map(one => (
              <Styled.Item key={one.url}>
                <Styled.Title href={one.url}>{one.title}</Styled.Title>
                <Styled.Name>{one.site}</Styled.Name>
                <Styled.Added>{one.addedAt.slice(0, 10)}</Styled.Added>
                {folders.length > 1 && (
                  <F.Move
                    value=""
                    disabled={busy}
                    aria-label={t('favourites.move')}
                    onChange={e => run({ action: 'move', article: one.articleId, folder: Number(e.target.value) })}
                  >
                    <option value="" disabled>
                      {t('favourites.move')}
                    </option>
                    {folders
                      .filter(f => f.id !== folder)
                      .map(f => (
                        <option key={f.id} value={f.id}>
                          {folderName(f)}
                        </option>
                      ))}
                  </F.Move>
                )}
              </Styled.Item>
            ))}
          </Styled.List>
          <Pager page={listing.page} pages={listing.pages} onPage={onPage} />
        </>
      )}
    </ProfilePage>
  )
}

export default Favourites
