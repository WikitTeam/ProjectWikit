import { callModule } from './modules'

export interface SearchModuleParams {
  q?: string
  author?: string
  tags?: string
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

export async function searchModule(params: SearchModuleParams) {
  return await callModule<SearchModuleResponse>({ module: 'search', method: 'search', params })
}
