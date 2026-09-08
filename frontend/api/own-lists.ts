import { wFetch } from '../util/fetch-util'

export interface RatingEntry {
  pageId: string
  title: string
  rate: number
  votedAt: string | null
}

export interface RatingListing {
  page: number
  pages: number
  total: number
  ratings: Array<RatingEntry>
}

export interface LikedPostEntry {
  postId: number
  name: string
  threadName: string
  url: string
  likedAt: string
}

export interface LikedPostListing {
  page: number
  pages: number
  total: number
  posts: Array<LikedPostEntry>
}

export async function getOwnRatings(page: number) {
  return await wFetch<RatingListing>(`/pw-api/ratings?page=${page}`)
}

export async function getOwnLikedPosts(page: number) {
  return await wFetch<LikedPostListing>(`/pw-api/liked-posts?page=${page}`)
}
