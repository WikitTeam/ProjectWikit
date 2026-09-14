import * as React from 'react'
import { HelmetProvider } from 'react-helmet-async'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from 'styled-components'
import ConfigContextProvider from '~reactive/config'
import { IConfigContext } from '~reactive/config/ConfigContext.types'
import Favourites from '~reactive/pages/favourites'
import LikedPosts from '~reactive/pages/liked-posts'
import Messages from '~reactive/pages/messages'
import Ratings from '~reactive/pages/ratings'
import Notifications from '~reactive/pages/notifications'
import { Paths } from '~reactive/paths'
import { SYSTEM_THEME } from '~reactive/theme/Theme.consts'

export default function ReactivePage() {
  const reactiveRoot: HTMLElement = document.querySelector('#reactive-root')!
  const config: IConfigContext = JSON.parse(reactiveRoot.dataset.config!)

  if (config.user.type !== 'normal') {
    window.location.href = `/-/login?to=${encodeURIComponent(window.location.href)}`
    return null
  }

  return (
    <HelmetProvider>
      <ConfigContextProvider config={config}>
        <ThemeProvider theme={SYSTEM_THEME}>
          <BrowserRouter basename="/-">
            <Routes>
              <Route path={`${Paths.notifications}/*`} element={<Notifications />} />
              <Route path={Paths.messages} element={<Messages />} />
              <Route path={`${Paths.messages}/:user_id`} element={<Messages />} />
              <Route path={Paths.favourites} element={<Favourites />} />
              <Route path={Paths.ratings} element={<Ratings />} />
              <Route path={Paths.likedPosts} element={<LikedPosts />} />
            </Routes>
          </BrowserRouter>
        </ThemeProvider>
      </ConfigContextProvider>
    </HelmetProvider>
  )
}
