import { t } from '~util/i18n'
import * as React from 'react'
import { useCallback, useState } from 'react'
import styled from 'styled-components'
import { favouriteArticle, unfavouriteArticle } from '../api/favourites'
import useConstCallback from '../util/const-callback'
import WikidotModal from '../util/wikidot-modal'

interface Props {
  pageId: string
  favourites: number
  favourited: boolean
  onClose: () => void
}

const Styles = styled.div`
  .w-favourite-star {
    font-size: 200%;
    text-decoration: none;
  }
`

const ArticleFavourite: React.FC<Props> = ({ pageId, favourites: initialCount, favourited: initialMine, onClose }) => {
  const [count, setCount] = useState(initialCount)
  const [mine, setMine] = useState(initialMine)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string>()

  const onToggle = useCallback(
    async (e: React.MouseEvent) => {
      e.preventDefault()
      if (saving) return
      setSaving(true)
      try {
        const result = mine ? await unfavouriteArticle(pageId) : await favouriteArticle(pageId)
        setMine(result.favourited)
        setCount(result.favourites)
      } catch (err: any) {
        setError(err.error || t('articles.favourite.failed'))
      } finally {
        setSaving(false)
      }
    },
    [mine, pageId, saving],
  )

  const onCancel = useConstCallback((e: React.MouseEvent) => {
    e.preventDefault()
    onClose()
  })

  return (
    <Styles>
      {error && (
        <WikidotModal buttons={[{ title: t('articles.favourite.dismiss'), onClick: () => setError(undefined) }]} isError>
          <p>
            <strong>{t('articles.favourite.error-label')}</strong> {error}
          </p>
        </WikidotModal>
      )}
      <h1>{t('articles.favourite.title')}</h1>
      <p>
        <a href="#" className="w-favourite-star" onClick={onToggle} title={t('articles.favourite.toggle')}>
          <i className={`${mine ? 'fas' : 'far'} fa-star`} />
        </a>{' '}
        <span className="w-favourite-count">{t('articles.favourite.count', { count })}</span>
      </p>
      <p>{t('articles.favourite.private')}</p>
      <div className="buttons alignleft">
        <input type="button" className="btn btn-danger" value={t('articles.favourite.close')} onClick={onCancel} />
      </div>
    </Styles>
  )
}

export default ArticleFavourite
