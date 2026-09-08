import { callModule } from './modules'
import { wFetch } from '../util/fetch-util'

export interface FavouriteState {
  pageId: string
  favourites: number
  favourited: boolean
}

export async function favouriteArticle(pageId: string) {
  return await callModule<FavouriteState>({ module: 'rate', method: 'favourite', pageId })
}

export async function unfavouriteArticle(pageId: string) {
  return await callModule<FavouriteState>({ module: 'rate', method: 'unfavourite', pageId })
}

export async function fetchFavouriteState(pageId: string) {
  return await callModule<FavouriteState>({ module: 'rate', method: 'get_favourites', pageId })
}

export interface FavouriteEntry {
  pageId: string
  title: string
  addedAt: string
}

export interface FavouriteListing {
  page: number
  pages: number
  total: number
  favourites: Array<FavouriteEntry>
}

export async function getFavourites(page: number) {
  return await wFetch<FavouriteListing>(`/pw-api/favourites?page=${page}`)
}
