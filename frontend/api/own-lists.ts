import { wFetch } from '../util/fetch-util'

export interface RatingEntry {
  pageId: string
  site: string
  url: string
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
  site: string
  url: string
  likedAt: string
}

export interface LikedPostListing {
  page: number
  pages: number
  total: number
  posts: Array<LikedPostEntry>
}

export interface TicketEntry {
  kind: 'ticket' | 'membershipapply' | 'report'
  id: number
  site: string
  url: string
  subject: string
  status: string
  reply: string
  createdAt: string
  reviewedAt: string | null
}

export interface TicketListing {
  page: number
  pages: number
  total: number
  tickets: Array<TicketEntry>
}

export async function getOwnTickets(page: number) {
  return await wFetch<TicketListing>(`/pw-api/my-tickets?page=${page}`)
}

export async function getOwnRatings(page: number) {
  return await wFetch<RatingListing>(`/pw-api/ratings?page=${page}`)
}

export async function getOwnLikedPosts(page: number) {
  return await wFetch<LikedPostListing>(`/pw-api/liked-posts?page=${page}`)
}
