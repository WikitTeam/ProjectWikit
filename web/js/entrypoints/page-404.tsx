import { t } from '~util/i18n'
import * as React from 'react'
import Trans from '~util/trans'
import { useState } from 'react'
import useConstCallback from '../util/const-callback'

interface Props {
  pageId?: string
  pathParams?: { [key: string]: string }
}

const Page404: React.FC<Props> = ({ pageId, pathParams }) => {
  const [isCreate, setIsCreate] = useState(!!pathParams?.['edit'])

  const onCreate = useConstCallback(e => {
    e.preventDefault()
    e.stopPropagation()
    setIsCreate(true)
    ;(window as any)._openNewEditor(() => {
      setIsCreate(false)
    })
  })

  if (!isCreate) {
    return (
      <>
        <p id="404-message">
          <Trans id="page-404.not-found" children={{ page: <em>{pageId}</em> }} />
        </p>
        <ul id="create-it-now-link">
          <li>
            <a href="#" onClick={onCreate}>
              {t('page-404.create-page')}
            </a>
          </li>
        </ul>
      </>
    )
  } else {
    return <></>
  }
}

export default Page404
