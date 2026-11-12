import { wFetch } from '../util/fetch-util'
import { callModule } from './modules'

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
  articleId: number
  pageId: string
  site: string
  url: string
  title: string
  addedAt: string
}

export interface FavouriteFolder {
  id: number
  name: string
  public: boolean
  count: number
}

export interface FavouriteListing {
  folders: Array<FavouriteFolder>
  folder: number
  page: number
  pages: number
  total: number
  favourites: Array<FavouriteEntry>
}

export async function getFavourites(folder: number, page: number) {
  return await wFetch<FavouriteListing>(`/pw-api/favourites?folder=${folder}&page=${page}`)
}

type FolderChange =
  | { action: 'create'; name: string; public: boolean }
  | { action: 'visibility'; folder: number; public: boolean }
  | { action: 'delete'; folder: number }
  | { action: 'move'; article: number; folder: number }

export async function changeFavourites(change: FolderChange) {
  return await wFetch(`/pw-api/favourites`, { method: 'POST', sendJson: true, body: change })
}
