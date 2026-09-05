import { callModule } from './modules'
import { UserData } from './user'

export interface PostLikeState {
  postId: number
  count: number
  liked: boolean
}

export interface PostLikesResponse {
  postId: number
  count: number
  page: number
  pages: number
  perPage: number
  users: UserData[]
}

export async function likePost(postId: number) {
  return await callModule<PostLikeState>({ module: 'forumpost', method: 'like', params: { postid: postId } })
}

export async function unlikePost(postId: number) {
  return await callModule<PostLikeState>({ module: 'forumpost', method: 'unlike', params: { postid: postId } })
}

export async function fetchPostLikes(postId: number, page: number) {
  return await callModule<PostLikesResponse>({
    module: 'forumpost',
    method: 'likes',
    params: { postid: postId, page },
  })
}
