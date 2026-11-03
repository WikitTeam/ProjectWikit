import { callModule } from './modules'

export interface SearchModuleParams {
  q?: string
  author?: string
  tags?: string
  category?: string
  datefrom?: string
  dateto?: string
  offset?: number
}

export interface SearchResultItem {
  title: string
  url: string
  excerpt: string
  words: string[]
  author: { name: string; url: string } | null
  tags: string[]
  comments: number
  createdAt: string | null
  updatedAt: string | null
  rating: string | null
}

export interface SearchModuleResponse {
  results: SearchResultItem[]
  hasMore: boolean
  total: number
}

export interface ForumResultItem {
  title: string
  post: string
  where: string
  url: string
  excerpt: string
  words: string[]
  author: { name: string; url: string } | null
  createdAt: string | null
}

export interface ForumSearchParams {
  q: string
  author: string
  category: string
  datefrom: string
  dateto: string
  offset: number
}

export async function searchForum(params: ForumSearchParams) {
  return await callModule<{ results: ForumResultItem[]; hasMore: boolean }>({ module: 'search', method: 'forum', params })
}

export interface ForumPlace {
  value: string
  name: string
  group: string
}

export async function forumPlaces() {
  return (await callModule<{ places: ForumPlace[] }>({ module: 'search', method: 'forum_places', params: {} })).places
}

export async function searchModule(params: SearchModuleParams) {
  return await callModule<SearchModuleResponse>({ module: 'search', method: 'search', params })
}

export type SuggestField = 'author' | 'tags' | 'category'

export interface SuggestItem {
  value: string
  label: string
}

export async function suggestSearch(field: SuggestField, q: string) {
  const resp = await callModule<{ items: SuggestItem[] }>({ module: 'search', method: 'suggest', params: { field, q } })
  return resp.items
}
